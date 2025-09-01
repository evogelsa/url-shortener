package main

import (
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/jxskiss/base62"
)

const (
	SAVED_LINKS   = "links.csv"
	SAVED_COUNTER = "counter.gob"
	API_KEY       = "token"
	API_KEY_ENV   = "ETHANS_LINK_API_KEY"
)

var (
	links   map[string]string
	counter int64
	apiKey  string
)

type ExtraData struct {
	Extra string
}

// init logging
func init() {
	f, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Could not create log file: %v", err)
		os.Exit(1)
	}

	log.SetOutput(f)
}

// load apiKey from environment variable or file
func init() {
	b, err := os.ReadFile(API_KEY)
	if err == nil {
		apiKey = strings.TrimSpace(string(b))
		log.Printf("loaded key from file: %s", apiKey)
	} else {
		log.Printf("error loading key from file: %v", err)
	}

	env := os.Getenv(API_KEY_ENV)
	if env != "" {
		apiKey = env
		log.Printf("loaded key from env: %s", apiKey)
	}
}

// check for saved links and load or create
func init() {
	_, err := os.Stat(SAVED_LINKS)
	if err == nil {
		// file already exists
		f, err := os.Open(SAVED_LINKS)
		if err != nil {
			log.Printf("Could not open file: %v", err)
		}
		defer f.Close()

		reader := csv.NewReader(f)
		rawLinks, err := reader.ReadAll()
		if err != nil {
			log.Printf("Could not decode links: %v", err)
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
			log.Printf("Could not create file: %v", err)
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
			log.Printf("Could not write links: %v", err)
			os.Exit(1)
		}
	}

	// add api to links so it cannot be used
	links["api"] = "inuse"
	links["api/create"] = "inuse"
}

// check for saved counter and load
func init() {
	_, err := os.Stat(SAVED_COUNTER)
	if err == nil {
		// file already exists
		f, err := os.Open(SAVED_COUNTER)
		if err != nil {
			log.Printf("Could not open file: %v", err)
			os.Exit(1)
		}
		defer f.Close()

		dec := gob.NewDecoder(f)
		err = dec.Decode(&counter)
		if err != nil {
			log.Printf("Could not decode counter: %v", err)
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
		log.Printf("Could not create file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	err = enc.Encode(counter)
	if err != nil {
		log.Printf("Could not encode counter: %v", err)
		os.Exit(1)
	}

	return s
}

func home(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("create.html"))
	extra := ExtraData{Extra: ""}

	err := tmpl.Execute(w, extra)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Could not parse form: %v", err)
		return
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

		err = tmpl.Execute(w, extra)
		if err != nil {
			log.Printf("Error executing template: %v", err)
		}
		return
	}

	_, err = url.ParseRequestURI(uri)
	if err != nil {
		tmpl := template.Must(template.ParseFiles("create.html"))
		extra := ExtraData{
			Extra: `<p style="color: #d50000">That is not a valid URL!<p>`,
		}

		err = tmpl.Execute(w, extra)
		if err != nil {
			log.Printf("Error executing template: %v", err)
		}
		return
	}

	links[id] = uri

	// save links
	f, err := os.OpenFile(SAVED_LINKS, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Could not create file: %v", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	err = writer.Write([]string{id, uri})
	if err != nil {
		log.Printf("Could not write link: %v", err)
		return
	}

	writer.Flush()
	if writer.Error() != nil {
		log.Printf("Could not write link: %v", err)
		return
	}

	link := fmt.Sprintf("https://ethans.link/%s", id)

	tmpl := template.Must(template.ParseFiles("create.html"))
	extra := ExtraData{
		Extra: `<label for="result">Your URL:</label>
				<input type="text" value="` + link + `" id="result">
				<button onclick="copyText()">Copy</button>`,
	}
	err = tmpl.Execute(w, extra)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func createAPI(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	uri := r.URL.Query().Get("url")
	id := r.URL.Query().Get("custom")

	if key != apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
		return
	}

	if id == "" {
		id = newID()
	}

	if _, exists := links[id]; exists {
		// alias in use
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("in use"))
		return
	}

	_, err := url.ParseRequestURI(uri)
	if err != nil {
		// not a valid url
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad url"))
		return
	}

	links[id] = uri

	// save links
	f, err := os.OpenFile(SAVED_LINKS, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Could not create file: %v", err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	err = writer.Write([]string{id, uri})
	if err != nil {
		log.Printf("Could not write link: %v", err)
		return
	}

	writer.Flush()
	if writer.Error() != nil {
		log.Printf("Could not write link: %v", err)
		return
	}

	link := fmt.Sprintf("https://ethans.link/%s", id)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(link))
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if uri, ok := links[id]; ok {
		http.Redirect(w, r, uri, http.StatusMovedPermanently)
	} else {
		log.Printf("ID not found in table: %s\n", id)
		http.NotFound(w, r)
	}
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", home).Methods("GET")
	router.HandleFunc("/", create).Methods("POST")
	router.HandleFunc("/api/create", createAPI).Methods("POST")
	router.HandleFunc("/{id}", redirect).Methods("GET")

	log.Fatal(http.ListenAndServe(":8085", router))
}
