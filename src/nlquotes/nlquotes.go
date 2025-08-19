package nlquotes

import (
	"MelvinBot/src/util"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	disc "github.com/bwmarrin/discordgo"
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

func FetchNLQuote(search string) (string, error) {
	url := fmt.Sprintf("https://nlquotes.com/api?search=%s&page=1&strict=false&channel=all&selectedMode=searchText&year=&sort=default&game=all",
		search)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	var apiResp struct {
		Data []NLEntry `json:"data"`
	}

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return "", nil
	}

	// Pick a random entry
	randomEntry := apiResp.Data[rand.Intn(len(apiResp.Data))]

	if len(randomEntry.Quotes) == 0 {
		return "", fmt.Errorf("no quotes in selected entry")
	}

	// Pick a random quote from the entry
	randomQuote := randomEntry.Quotes[rand.Intn(len(randomEntry.Quotes))]

	parsedTimestamp, err := strconv.ParseFloat(randomQuote.TimestampStart, 64)
	if err != nil {
		return "", fmt.Errorf("had an issue parsing the timestamp: %w", err)
	}
	youtubeLink := fmt.Sprintf("https://youtu.be/%s/?t=%d", randomEntry.VideoID, int(parsedTimestamp))
	hyperlink := fmt.Sprintf("[%s](%s)", randomQuote.Text, youtubeLink)

	return hyperlink, nil
}

func RandomNLQuote() (string, error) {
	url := "https://nlquotes.com/api/random"

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	// The API returns a top-level "quotes" array
	var apiResp struct {
		Quotes []NLEntry `json:"quotes"`
	}

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if len(apiResp.Quotes) == 0 {
		return "", fmt.Errorf("no entries returned from API")
	}

	// Pick a random entry
	randomEntry := apiResp.Quotes[rand.Intn(len(apiResp.Quotes))]

	if len(randomEntry.Quotes) == 0 {
		return "", fmt.Errorf("no quotes in selected entry")
	}

	// Pick a random quote from the entry
	randomQuote := randomEntry.Quotes[rand.Intn(len(randomEntry.Quotes))]

	parsedTimestamp, err := strconv.ParseFloat(randomQuote.TimestampStart, 64)
	if err != nil {
		return "", fmt.Errorf("had an issue parsing the timestamp: %w", err)
	}
	youtubeLink := fmt.Sprintf("https://youtu.be/%s/?t=%d", randomEntry.VideoID, int(parsedTimestamp))
	hyperlink := fmt.Sprintf("[%s](%s)", randomQuote.Text, youtubeLink)

	return hyperlink, nil
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
