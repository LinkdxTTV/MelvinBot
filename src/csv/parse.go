package parse

import (
	"MelvinBot/src/quotes"
	"encoding/csv"
	"fmt"
	"os"
)

func ParseAndDedupCsv() ([]quotes.Quote, error) {
	var allQuotes []quotes.Quote
	csvFile, err := os.Open("/home/nelly/apps/bot/parsed_quotes.csv")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	var person, quote string
	quoteExistsMap := make(map[string]bool)
	csvRaw, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for _, row := range csvRaw {
		person = row[0]
		quote = row[1]
		if _, exists := quoteExistsMap[quote]; exists {
			continue
		} else {
			quoteExistsMap[quote] = true
			allQuotes = append(allQuotes, quotes.Quote{
				Author: person,
				Quote:  quote,
			})
		}
	}
	return allQuotes, err
}
