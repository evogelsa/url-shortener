package main

import (
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/jxskiss/base62"
)

const (
	SAVED_LINKS   = "links.csv"
	SAVED_COUNTER = "counter.gob"
)

var (
	links   map[string]string
	counter int64
)

type ExtraData struct {
	Extra string
}

// check for saved links and load or create
func init() {
	_, err := os.Stat(SAVED_LINKS)
	if err == nil {
		// file already exists
		f, err := os.Open(SAVED_LINKS)
		if err != nil {
			fmt.Printf("Could not open file: %v", err)
			os.Exit(1)
		}
		defer f.Close()

		reader := csv.NewReader(f)
		rawLinks, err := reader.ReadAll()
		if err != nil {
			fmt.Printf("Could not decode links: %v", err)
			os.Exit(1)
		}

		links = make(map[string]string)
		for _, link := range rawLinks {
			links[link[0]] = link[1]
		}

	} else {
		// make new map and file
		links = make(map[string]string)

		// create new file to save links to
		f, err := os.Create(SAVED_LINKS)
		if err != nil {
			fmt.Printf("Could not create file: %v", err)
			os.Exit(1)
		}
		defer f.Close()

		writer := csv.NewWriter(f)

		var rawLinks [][]string
		for key, value := range links {
			rawLinks = append(rawLinks, []string{key, value})
		}

		err = writer.WriteAll(rawLinks)
		if err != nil {
			fmt.Printf("Could not write links: %v", err)
			os.Exit(1)
		}
	}
}

// check for saved counter and load
func init() {
	_, err := os.Stat(SAVED_COUNTER)
	if err == nil {
		// file already exists
		f, err := os.Open(SAVED_COUNTER)
		if err != nil {
			fmt.Printf("Could not open file: %v", err)
			os.Exit(1)
		}
		defer f.Close()

		dec := gob.NewDecoder(f)
		err = dec.Decode(&counter)
		if err != nil {
			fmt.Printf("Could not decode counter: %v", err)
			os.Exit(1)
		}
	}
}

// creates new id and saves it
func newID() string {
	num := counter
	bytes := base62.FormatInt(num)
	s := base62.EncodeToString(bytes)

	// incr counter
	counter++

	// save
	f, err := os.Create(SAVED_COUNTER)
	if err != nil {
		fmt.Printf("Could not create file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(counter)
	if err != nil {
		fmt.Printf("Could not encode counter: %v", err)
		os.Exit(1)
	}

	return s
}

func home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("create.html"))
	extra := ExtraData{Extra: ""}

	tmpl.Execute(w, extra)
}

func create(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Could not parse form: %v", err)
		os.Exit(1)
	}

	uri := r.Form.Get("url")
	id := r.Form.Get("custom")

	if id == "" {
		id = newID()
	}

	if _, exists := links[id]; exists {
		tmpl := template.Must(template.ParseFiles("create.html"))
		extra := ExtraData{
			Extra: `<p style="color: #d50000">Sorry, that alias is in use!<p>`,
		}

		tmpl.Execute(w, extra)
		return
	}

	_, err = url.ParseRequestURI(uri)
	if err != nil {
		tmpl := template.Must(template.ParseFiles("create.html"))
		extra := ExtraData{
			Extra: `<p style="color: #d50000">That is not a valid URL!<p>`,
		}

		tmpl.Execute(w, extra)
		return
	}

	links[id] = uri

	// save links
	f, err := os.Create(SAVED_LINKS)
	if err != nil {
		fmt.Printf("Could not create file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	var rawLinks [][]string
	for key, value := range links {
		rawLinks = append(rawLinks, []string{key, value})
	}

	err = writer.WriteAll(rawLinks)
	if err != nil {
		fmt.Printf("Could not write links: %v", err)
		os.Exit(1)
	}

	link := fmt.Sprintf("https://ethans.link/%s", id)

	tmpl := template.Must(template.ParseFiles("create.html"))
	extra := ExtraData{
		Extra: `<label for="result">Your URL:</label>
				<input type="text" value="` + link + `" id="result">
				<button onclick="copyText()">Copy</button>`,
	}
	tmpl.Execute(w, extra)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if uri, ok := links[id]; ok {
		http.Redirect(w, r, uri, http.StatusMovedPermanently)
	} else {
		fmt.Printf("ID not found in table: %s\n", id)
		return
	}
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", home).Methods("GET")
	router.HandleFunc("/", create).Methods("POST")
	router.HandleFunc("/{id}", redirect).Methods("GET")

	log.Fatal(http.ListenAndServe(":8085", router))
}
