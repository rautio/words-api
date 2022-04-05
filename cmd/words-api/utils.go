package main

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"strings"
)
func ReadCsvFile(filePath string) [][]string {
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

func ReadTxtFileByLine(filePath string) []string {
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
func getPort() string {
  p := os.Getenv("PORT")
  if p != "" {
    return ":" + p
  }
  return ":9000"
}