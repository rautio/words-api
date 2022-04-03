package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Word struct {
	Word             string `json:"word"`
	commonalityScore int    `json: "commonalityScore`
}

func main() {
	// Seeding to randomize words by time
	rand.Seed(time.Now().UTC().UnixNano())
	// Read the main words file
	file, err := os.Open("../words-api/assets/words.txt")
	if err != nil {
			log.Fatal(err)
	}
	defer func() {
			if err = file.Close(); err != nil {
					log.Fatal(err)
			}
	}()
	rawWords, err := ioutil.ReadAll(file)
	words := strings.Split(string(rawWords), "\n")
	// Parse it into separate word lengths to power the word length API later


	wordHandler := func(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
		word := vars["word"]
		for _, curWord := range words {
			if curWord == word {
				// io.WriteString(w, curWord)
				w.Header().Set("Content-Type", "application/json")
				result := Word{curWord, 0}
				w.Header().Set("Content-Type", "application/json")
				jsonResponse, _ := json.Marshal(result)
				w.Write(jsonResponse)
				return
			}
		}
		http.Error(w, "None found", http.StatusBadRequest)
	}

	wordsHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, strings.Join(words[:], "\n"))
	}

	randomWordHandler := func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		_, hasLength := req.Form["length"]

		// shuffle the order of words 
		dest := make([]string, len(words))
		perm := rand.Perm(len(words))
		for i, v := range perm {
				dest[v] = string(words[i])
		}
		randomWord := Word{dest[0], 0}
		if hasLength {
			length, err := strconv.Atoi(req.URL.Query().Get("length"))
			if err != nil {
					log.Fatal(err)
			}
			for _, s := range dest {
				if len(s) == length {
					randomWord = Word{s, 0}
					break;
				}
			}
		}
		jsonResponse, jsonError := json.Marshal(randomWord)
		if jsonError != nil {
		  fmt.Println("Unable to encode JSON")
		}
    w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	}
	
	
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/random", randomWordHandler).Methods("GET","OPTIONS")
		log.Println("Listing for requests at http://localhost:9001/random")
	router.HandleFunc("/word/{word}", wordHandler).Methods("GET","OPTIONS")
		log.Println("Listing for requests at http://localhost:9001/word")
	router.HandleFunc("/words", wordsHandler).Methods("GET","OPTIONS")
		log.Println("Listing for requests at http://localhost:9001/words")
	http.Handle("/", router)
	http.ListenAndServe(":9001", handlers.CORS()(router))
}