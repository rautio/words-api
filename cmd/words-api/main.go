package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
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
	// TODO: Create a map of word: <Word> for easier lookups later
	words := ReadTxtFileByLine("../../assets/words.txt")
	// Read frequency records
	// TODO: Convert frequency count to a frequency score and return in the json data
	mostFreqWords := ReadCsvFile("../../assets/unigram_freq.csv")
	// Sort by the frequency count
	sort.Slice(mostFreqWords, func(i, j int) bool {
		r1, _ := strconv.Atoi(mostFreqWords[i][1])
		r2, _ :=  strconv.Atoi(mostFreqWords[j][1])
		return r1 > r2
	})


	/**
	 	* Return info about a specific word
		*    /word/<:word>
		* 
		* Query Params: 
		* 	length=<int> : number of characters the random word should contain
		*/
	wordHandler := func(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
		word := vars["word"]
		// Iterate through all known words and look for the matching one
		for _, curWord := range words {
			if curWord == word {
				result := Word{curWord, 0}
				w.Header().Set("Content-Type", "application/json")
				jsonResponse, _ := json.Marshal(result)
				w.Write(jsonResponse)
				return
			}
		}
		// If there was no match above then it is an unknown word
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "None Found", http.StatusBadRequest)
		return
	}

	wordsHandler := func(w http.ResponseWriter, req *http.Request) {
		// TODO: Return in an array of objects format
		io.WriteString(w, strings.Join(words[:], "\n"))
		return
	}

	/**
	 	* Return a single randomized word.
		* 
		* Query Params: 
		* 	length=<int> : number of characters the random word should contain
		*/
	randomWordHandler := func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		_, hasLength := req.Form["length"]
		length, _ := strconv.Atoi(req.URL.Query().Get("length"))

		// Use the top X number of most common words
		// The higher the number the higher the chance of returning uncommon words
		wordLimit := 1000
		wordsToChoose := make([]string, wordLimit)	
    i := 0
		added := 0
		// Calculate the most frequent words that match the length (if provided)
		for added < wordLimit && i < len(mostFreqWords) {
			if (hasLength) {
				if (len(mostFreqWords[i][0]) == length) {
					wordsToChoose[added] = mostFreqWords[i][0]
					added++
				}
			} else {
				wordsToChoose[added] = mostFreqWords[i][0]
				added++
			}
			i++
		}
		// Choose a word at random from the most frequent sub-list
		randIdx := rand.Intn(wordLimit)
		randomWord := wordsToChoose[randIdx]
		result := Word{randomWord, 0}
		jsonResponse, jsonError := json.Marshal(result)
		if jsonError != nil {
		  fmt.Println("Unable to encode JSON")
		}
    w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}
	
	// Router setup
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/random", randomWordHandler).Methods("GET","OPTIONS")
	log.Println("Listening for requests at http://localhost:9001/random")

	router.HandleFunc("/word/{word}", wordHandler).Methods("GET","OPTIONS")
	log.Println("Listening for requests at http://localhost:9001/word")

	router.HandleFunc("/word", wordsHandler).Methods("GET","OPTIONS")
	log.Println("Listening for requests at http://localhost:9001/words")

	// TODO: Return a API doc page w/ examples like type ahead
	http.Handle("/", router)
	http.ListenAndServe(":9001", handlers.CORS()(router))
}