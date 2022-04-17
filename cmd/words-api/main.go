package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Word struct {
	Word             string `json:"word"`
}

func main() {
	// Connect to DB
  db, err := sql.Open("postgres", getDatabaseUrl())
  if err != nil {
    log.Fatal(err)
  }
	// Need the uuid extension if it doesn't exist
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	// Word Guess table
	db.Exec(`CREATE TABLE IF NOT EXISTS guessr (
		id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
		word VARCHAR (20) NOT NULL,
		source VARCHAR(20),
		create_time TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_DATE
	)`)
	// Session table
	db.Exec(`CREATE TABLE IF NOT EXISTS guessr_session (
		id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
		guessr_id UUID NOT NULL,
		correct BOOLEAN,
		guesses JSON,
		word VARCHAR (20) NOT NULL,
		create_time TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_DATE
	)`)

	pingErr := db.Ping()
	if pingErr != nil {
		log.Println(pingErr)
    log.Fatal(pingErr)
	}

	defer db.Close()
	// Seeding to randomize words by time
	rand.Seed(time.Now().UTC().UnixNano())
	// Read the main words file
	// TODO: Create a map of word: <Word> for easier lookups later
	absPath, _ := filepath.Abs("assets/words.txt")
	words := ReadTxtFileByLine(absPath)
	// Read frequency records
	// TODO: Convert frequency count to a frequency score and return in the json data
	absPathCsv, _ := filepath.Abs("assets/unigram_freq.csv")
	mostFreqWords := ReadCsvFile(absPathCsv)
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
				result := Word{curWord}
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
		result := Word{randomWord}
		jsonResponse, jsonError := json.Marshal(result)
		if jsonError != nil {
		  fmt.Println("Unable to encode JSON")
		}
    w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}

	createGuessrHandler := func(w http.ResponseWriter, req *http.Request) {
		var postBody map[string]interface{}
		decoder := json.NewDecoder(req.Body)
		decodePostErr := decoder.Decode(&postBody)
		if err != nil {
			log.Println(decodePostErr)
			panic(decodePostErr)
		}
		if word, ok := postBody["word"]; ok {
			// Connect to DB
			var lastInsertId []uint8; // uuid v4 format
			source := "n/a";
			if src, ok := postBody["source"]; ok {
				source = fmt.Sprintf("%v", src)
			}
			db, _ := sql.Open("postgres", getDatabaseUrl())
			err := db.QueryRow(`INSERT INTO guessr (word, source)
			VALUES ($1, $2) RETURNING id`, word, source).Scan(&lastInsertId)
			defer db.Close()
			CheckError(err)
			result := map[string]interface{}{ "id": string([]byte(lastInsertId)) }
			jsonResponse, jsonError := json.Marshal(result)
			if jsonError != nil {
				fmt.Println("Unable to encode JSON")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
			return
		}
		// Missing word in the post body
		log.Println("No word provided")
		log.Println(req.FormValue("word"))
		log.Println(req.Form)
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "word is required", http.StatusBadRequest)
		return
	}


	storeSessionHandler := func(w http.ResponseWriter, req *http.Request) {
		var postBody map[string]interface{}
		decoder := json.NewDecoder(req.Body)
		decodePostErr := decoder.Decode(&postBody)
		if decodePostErr != nil {
			log.Println(decodePostErr)
			panic(decodePostErr)
		}
		guessrId := postBody["guessr_id"]
		guesses := postBody["guesses"]
		correct := postBody["correct"]
		// Connect to DB
		var lastInsertId []uint8; // uuid v4 format
		db, _ := sql.Open("postgres", getDatabaseUrl())
		err := db.QueryRow(`INSERT INTO guessr_session (guessr_id, guesses, correct)
		VALUES ($1, $2, $3) RETURNING id`, guessrId, guesses, correct).Scan(&lastInsertId)
		defer db.Close()
		CheckError(err)
		result := map[string]interface{}{ "id": string([]byte(lastInsertId)) }
		jsonResponse, jsonError := json.Marshal(result)
		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}

	getGuessrHandler := func(w http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
		id := vars["id"]
		uId, uuidErr := uuid.Parse(id)
		if uuidErr != nil {
			log.Println(uuidErr)
			w.WriteHeader(http.StatusBadRequest)
			http.Error(w, "Invalid id", http.StatusBadRequest)
			return
		}
		var resultId string
		var word string
		var source string
		// Connect to DB
		db, _ := sql.Open("postgres", getDatabaseUrl())
		err := db.QueryRow(`SELECT id, word, source FROM guessr WHERE id=$1;`, uId).Scan(&resultId, &word, &source)
		defer db.Close()
		if err != nil {
			log.Println(err)
			// If there was no match above then it is an unknown word
			w.WriteHeader(http.StatusBadRequest)
			http.Error(w, "None Found", http.StatusBadRequest)
			return
		}
		result := map[string]interface{}{ "id": resultId, "word": word, "source": source }
		jsonResponse, jsonError := json.Marshal(result)
		if jsonError != nil {
			log.Println(jsonError)
		  fmt.Println("Unable to encode JSON")
		}
    w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}
	
	// Router setup
	router := mux.NewRouter().StrictSlash(true)

	port := getPort()

	router.HandleFunc("/random", randomWordHandler).Methods("GET","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/random", port))

	router.HandleFunc("/word/{word}", wordHandler).Methods("GET","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/word/{word}", port))

	router.HandleFunc("/word", wordsHandler).Methods("GET","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/word", port))

	router.HandleFunc("/guessr", createGuessrHandler).Methods("POST","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/guessr", port))

	router.HandleFunc("/guessr-session", storeSessionHandler).Methods("POST","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/guessr-session", port))

	router.HandleFunc("/guessr/{id}", getGuessrHandler).Methods("GET","OPTIONS")
	log.Println(fmt.Sprintf("Listening for requests at http://localhost%s/guessr/{id}", port))


	// TODO: Return a API doc page w/ examples like type ahead
	http.Handle("/", router)
	http.ListenAndServe(port, handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization" }), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),handlers.AllowedOrigins([]string{"*"}))(router))
}


func CheckError(err error) {
	if err != nil {
			panic(err)
	}
}