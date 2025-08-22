package nlquotes

import (
	"MelvinBot/src/util"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	disc "github.com/bwmarrin/discordgo"
)

const (
	EntriesPerPage int = 10
)

type NLQuote struct {
	Text           string `json:"text"`
	TimestampStart string `json:"timestamp_start"`
}

type NLEntry struct {
	VideoID       string    `json:"video_id"`
	Title         string    `json:"title"`
	UploadDate    string    `json:"upload_date"`
	ChannelSource string    `json:"channel_source"`
	Quotes        []NLQuote `json:"quotes"`
}

func convertBoldToMarkdown(input string) string {
	// Replace <b> with **
	re := regexp.MustCompile(`(?i)<\/?b>`)
	return re.ReplaceAllStringFunc(input, func(tag string) string {
		if tag[1] == '/' { // closing tag
			return "**"
		}
		return "**"
	})
}

func queryNLAPI(endpoint string, params map[string]string, response any) error {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://nlquotes.com/api/%s", endpoint), nil)
	if err != nil {
		return fmt.Errorf("Failed to parse URL")
	}

	query := request.URL.Query()
	for key := range params {
		query.Add(key, params[key])
	}
	request.URL.RawQuery = query.Encode()

	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("failed to read API response: %w", err)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	return nil
}

func formatNLQuote(entry NLEntry, quote NLQuote) (string, error) {
	cleanText := convertBoldToMarkdown(quote.Text)

	parsedTimestamp, err := strconv.ParseFloat(quote.TimestampStart, 64)
	if err != nil {
		return "", fmt.Errorf("had an issue parsing the timestamp: %w", err)
	}
	youtubeLink := fmt.Sprintf("https://youtu.be/%s/?t=%d", entry.VideoID, int(parsedTimestamp))

	// Parse the timestamp using the RFC3339 layout
	uploadDate, err := time.Parse(time.RFC3339, entry.UploadDate)
	if err != nil {
		return "", fmt.Errorf("Error parsing upload date: %w", err)
	}

	quoteOffset := time.Duration(parsedTimestamp * float64(time.Second))

	hyperlink := fmt.Sprintf("[%s @ %s](%s)", uploadDate.Format("January 2, 2006"), quoteOffset.Round(time.Second), youtubeLink)
	finalMessage := fmt.Sprintf("%s\n%s", cleanText, hyperlink)

	return finalMessage, nil
}

func formatRandomNLEntry(entries []NLEntry) (string, error) {
	// Pick a random entry
	randomEntry := entries[rand.Intn(len(entries))]

	return formatRandomNLQuote(randomEntry)
}

func formatRandomNLQuote(entry NLEntry) (string, error) {
	if len(entry.Quotes) == 0 {
		return "", fmt.Errorf("no quotes in selected entry")
	}

	// Pick a random quote from the entry
	randomQuote := entry.Quotes[rand.Intn(len(entry.Quotes))]

	message, err := formatNLQuote(entry, randomQuote)

	if err != nil {
		return "", fmt.Errorf("failed to format quote: %w", err)
	}

	return message, nil
}

func FetchNLQuote(search string) (string, error) {
	// So the API returns a series of entries, each of which can have multiple
	// quotes. 10 entries are returned per page.
	//
	// We want to pick a random quote, but we can't predict which page a given
	// quote will be present on.
	//
	// 	We _can_ guarantee which page a given _entry_ will be on
	// 	(`entryIndex/10 + 1`), so we'll settle for picking a random entry and
	// 	then selecting a random quote from that entry. This will bias results
	// 	_away_ from quotes in entries with many quotes, but that's fine.

	var apiResp struct {
		Data        []NLEntry `json:"data"`
		Total       int       `json:"total"`
		TotalQuotes int       `json:"totalQuotes"`
	}

	headers := map[string]string{
		"search":       search,
		"page":         "1",
		"strict":       "false",
		"channel":      "all",
		"selectedMode": "searchText",
		"year":         "",
		"sort":         "default",
		"game":         "all",
	}

	err := queryNLAPI("", headers, &apiResp)

	if err != nil {
		return "", err
	}

	entryIndex := rand.Intn(apiResp.Total)

	if entryIndex >= EntriesPerPage {
		page := entryIndex/EntriesPerPage + 1
		entryIndex = entryIndex % EntriesPerPage
		headers["page"] = strconv.Itoa(page)
		err = queryNLAPI("", headers, &apiResp)

		if err != nil {
			return "", err
		}
	}

	if len(apiResp.Data) == 0 {
		return "", nil
	}

	return formatRandomNLQuote(apiResp.Data[entryIndex])
}

func RandomNLQuote() (string, error) {
	// The API returns a top-level "quotes" array
	var apiResp struct {
		Quotes []NLEntry `json:"quotes"`
	}

	err := queryNLAPI("random", map[string]string{}, &apiResp)

	if err != nil {
		return "", err
	}

	if len(apiResp.Quotes) == 0 {
		return "", fmt.Errorf("no entries returned from API")
	}

	return formatRandomNLEntry(apiResp.Quotes)
}

func HandleNLQuote(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}
	if !strings.HasPrefix(m.Content, "!nlquote") {
		return
	}

	// Remove the command prefix and trim whitespace
	searchTerm := strings.TrimSpace(strings.TrimPrefix(m.Content, "!nlquote"))

	var quote string
	var err error

	if searchTerm == "" {
		// Case 1: No search term, fetch a completely random quote
		quote, err = RandomNLQuote()
		if err != nil {
			util.SendSelfDestructingMessage(s, m.ChannelID, "couldn't pull a random quote sorry, maybe the API is down?", 5*time.Second)
			return
		}
	} else {
		// Case 2: Search term provided, fetch matching quotes
		quote, err = FetchNLQuote(searchTerm)
		if quote == "" && err == nil { // special case where the response array had length 0
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("shockingly NL has never said '%s'", searchTerm))
		}
		if err != nil {
			util.SendSelfDestructingMessage(s, m.ChannelID, "sorry got an error trying that", 5*time.Second)
			return
		}
	}

	// Send the quote to the channel
	s.ChannelMessageSend(m.ChannelID, quote)
}
