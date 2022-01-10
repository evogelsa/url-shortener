package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/mux"
	"github.com/jxskiss/base62"
)

const (
	SAVED_LINKS   = "links.gob"
	SAVED_COUNTER = "counter.gob"
)

var (
	links   map[string]string
	counter int64
)

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

		dec := gob.NewDecoder(f)
		err = dec.Decode(&links)
		if err != nil {
			fmt.Printf("Could not decode links: %v", err)
			os.Exit(1)
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

		enc := gob.NewEncoder(f)
		err = enc.Encode(links)
		if err != nil {
			fmt.Printf("Could not encode links: %v", err)
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
	content := `<!doctype html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
		<link rel="stylesheet" href="/static/styles.css">
	  </head>
	  <body>
		<h1 id="shorten-url">Shorten URL</h1>
		<div>
		<form action="/" method="post" name="shorten" autocomplete="off">
		<p><label for="url">URL</label> <input type="text" id="url" name="url" width="100%" margin-top="6px" margin-bottom="16px" required></p>
		<p><label for="custom">Custom alias</label> <input type="text" id="custom" name="custom" width="100%" margin-top="6px" margin-bottom="16px"></p>
		<input type="submit" name="send" value="Submit">
		</form>
		</div>
	  </body>
	</html>
	`

	fmt.Fprint(w, content)
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
		content := `<!doctype html>
		<html lang="en">
		  <head>
			<meta charset="utf-8">
			<link rel="stylesheet" href="/static/styles.css">
		  </head>
		  <body>
			<h1 id="shorten-url">Shorten URL</h1>
			<div>
			<form action="/" method="post" name="shorten" autocomplete="off">
			<p><label for="url">URL</label> <input type="text" id="url" name="url" width="100%" margin-top="6px" margin-bottom="16px" required></p>
			<p><label for="custom">Custom alias</label> <input type="text" id="custom" name="custom" width="100%" margin-top="6px" margin-bottom="16px"></p>
			<input type="submit" name="send" value="Submit">
			</form>
			</div>
			<div>
			Alias already in use
			</div>
		  </body>
		</html>`

		fmt.Fprintf(w, content, id)
		return
	}

	_, err = url.ParseRequestURI(uri)
	if err != nil {
		content := `<!doctype html>
		<html lang="en">
		  <head>
			<meta charset="utf-8">
			<link rel="stylesheet" href="/static/styles.css">
		  </head>
		  <body>
			<h1 id="shorten-url">Shorten URL</h1>
			<div>
			<form action="/" method="post" name="shorten" autocomplete="off">
			<p><label for="url">URL</label> <input type="text" id="url" name="url" width="100%" margin-top="6px" margin-bottom="16px" required></p>
			<p><label for="custom">Custom alias</label> <input type="text" id="custom" name="custom" width="100%" margin-top="6px" margin-bottom="16px"></p>
			<input type="submit" name="send" value="Submit">
			</form>
			</div>
			<div>
			Not a valid URL
			</div>
		  </body>
		</html>`

		fmt.Fprintf(w, content, id)
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

	enc := gob.NewEncoder(f)
	err = enc.Encode(links)
	if err != nil {
		fmt.Printf("Could not encode links: %v", err)
		os.Exit(1)
	}

	link := fmt.Sprintf("https://ethans.link/%s", id)
	content := `<!doctype html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
		<link rel="stylesheet" href="/static/styles.css">
	  </head>
	  <body>
		<h1 id="shorten-url">Shorten URL</h1>
		<div>
		<form action="/" method="post" name="shorten" autocomplete="off">
		<p><label for="url">URL</label> <input type="text" id="url" name="url" width="100%" margin-top="6px" margin-bottom="16px" required></p>
		<p><label for="custom">Custom alias</label> <input type="text" id="custom" name="custom" width="100%" margin-top="6px" margin-bottom="16px"></p>
		<input type="submit" name="send" value="Submit">
		</form>
		</div>
		<div>
		URL created: ` + link +
		`</div>
	  </body>
	</html>`

	fmt.Fprintf(w, content, id)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if uri, ok := links[id]; ok {
		http.Redirect(w, r, uri, http.StatusMovedPermanently)
	} else {
		fmt.Printf("URL not found in table\n")
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
