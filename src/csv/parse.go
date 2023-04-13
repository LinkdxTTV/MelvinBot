package parse

import (
	"encoding/csv"
	"fmt"
	"os"
)

func ParseAndDedupCsv() ([]string, error) {
	var allQuotes []string
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
		if _, exists := quoteExistsMap[person]; exists {
			continue
		} else {
			quoteExistsMap[person] = true
			allQuotes = append(allQuotes, fmt.Sprintf("```%v```\n~%v", quote, person))
		}
	}
	return allQuotes, err
}
