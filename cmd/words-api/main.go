package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
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


func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
			log.Fatal("Unable to read input file " + filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
			log.Fatal("Unable to parse file as CSV for " + filePath, err)
	}

	return records
}

func readWordsTxtFile(filePath string) []string {
	file, err := os.Open(filePath)
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
	return words

}

func main() {
	// Seeding to randomize words by time
	rand.Seed(time.Now().UTC().UnixNano())
	// Read the main words file
	words := readWordsTxtFile("../words-api/assets/words.txt")
	// Read frequency records
	mostFreqWords := readCsvFile("../words-api/assets/unigram_freq.csv")
	sort.Slice(mostFreqWords, func(i, j int) bool {
		r1, _ := strconv.Atoi(mostFreqWords[i][1])
		r2, _ :=  strconv.Atoi(mostFreqWords[j][1])
		return r1 > r2
	})
	wordHandler := func(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
		word := vars["word"]
		for _, curWord := range words {
			if curWord == word {
				result := Word{curWord, 0}
				w.Header().Set("Content-Type", "application/json")
				jsonResponse, _ := json.Marshal(result)
				w.Write(jsonResponse)
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "None Found", http.StatusBadRequest)
		return
	}

	wordsHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, strings.Join(words[:], "\n"))
		return
	}

	randomWordHandler := func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		_, hasLength := req.Form["length"]
		length, _ := strconv.Atoi(req.URL.Query().Get("length"))

		// Get the top 10,000 most common words for the length
		wordLimit := 1000
		wordsToChoose := make([]string, wordLimit)	
    i := 0
		added := 0
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
		// Choose a word at random from the most frequent list
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