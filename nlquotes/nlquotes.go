package nlquotes

import (
  "encoding/json"
  "fmt"
  "io"
  "math/rand"
  "net/http"
  "strings"
  "time"

  disc "github.com/bwmarrin/discordgo"
)

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
        Total int      `json:"total"`
        TotalQuotes int      `json:"totalQuotes"`
        QueryTime int      `json:"queryTime"`
        TotalTime int      `json:"totalTime"`
    }

    err = json.Unmarshal(body, &apiResp)
    if err != nil {
        return "", fmt.Errorf("failed to unmarshal API response: %w", err)
    }

    if len(apiResp.Data) == 0 {
        return "", fmt.Errorf("no entries returned from API")
    }

    // Pick a random entry
    rand.Seed(time.Now().UnixNano())
    randomEntry := apiResp.Data[rand.Intn(len(apiResp.Data))]

    if len(randomEntry.Quotes) == 0 {
        return "", fmt.Errorf("no quotes in selected entry")
    }

    // Pick a random quote from the entry
    randomQuote := randomEntry.Quotes[rand.Intn(len(randomEntry.Quotes))]

    return randomQuote.Text, nil
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

    // Pick a random NLEntry
    rand.Seed(time.Now().UnixNano())
    randomEntry := apiResp.Quotes[rand.Intn(len(apiResp.Quotes))]

    if len(randomEntry.Quotes) == 0 {
        return "", fmt.Errorf("no quotes in selected entry")
    }

    // Pick a random quote from the entry
    randomQuote := randomEntry.Quotes[rand.Intn(len(randomEntry.Quotes))]

    return randomQuote.Text, nil
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
            s.ChannelMessageSend(m.ChannelID, "Error fetching random quote 😔, API down?")
            return
        }
    } else {
        // Case 2: Search term provided, fetch matching quotes
        quote, err = FetchNLQuote(searchTerm)
        if err != nil {
            s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("shockingly NL has never said '%s'", searchTerm))
            return
        }
    }

    // Send the quote to the channel
    s.ChannelMessageSend(m.ChannelID, quote)
}
