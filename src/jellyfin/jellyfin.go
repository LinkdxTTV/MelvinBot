package jellyfin

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var JellyfinUpdateChannels = []string{"1203552160177061989", "1203541512542093383"}

// jellyuserid and jellyapikey should be loaded into the .env file in the main routine

type JellyUpdater struct {
	userID         string
	apiKey         string
	discordSession *discordgo.Session
}

type JellyMedia struct {
	Name               string
	DateCreated        time.Time
	DateLastMediaAdded time.Time
	Type               string
	ChildCount         int
	ProductionYear     int
}

func NewJellyUpdater(disc *discordgo.Session) *JellyUpdater {
	userID := os.Getenv("jellyuserid")
	apiKey := os.Getenv("jellyapikey")

	JellyUpdater := &JellyUpdater{
		userID:         userID,
		apiKey:         apiKey,
		discordSession: disc,
	}

	return JellyUpdater
}

// We absolutely assume jellyfin is running locally at http://localhost:8096/jelly btw
func (j *JellyUpdater) GetRecentMediaSince(timeSince time.Time) ([]JellyMedia, error) {
	recentMediaEndpoint := fmt.Sprintf("http://localhost:8096/jelly/Users/%s/Items/Latest?fields=DateLastMediaAdded,DateCreated&enableImages=false&enableUserData=false", j.userID)

	url, err := url.Parse(recentMediaEndpoint)
	if err != nil {
		log.Println("failed to parse jellyfin recent media url", err)
		return nil, err
	}

	client := http.Client{}
	resp, err := client.Do(
		&http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: http.Header{
				"Authorization": []string{fmt.Sprintf("MediaBrowser Client=\"Jellyfin Web\", Device=\"Firefox\", DeviceId=\"abcdefg\", Version=\"10.7.6\", Token=\"%s\"", j.apiKey)},
			},
		},
	)
	if err != nil {
		log.Println("failed to make http req", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Println("failed to make http req, got invalid status code", resp.StatusCode)
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("could not read response body", err)
		return nil, err
	}

	medias := []JellyMedia{}
	err = json.Unmarshal(b, &medias)
	if err != nil {
		return nil, err
	}

	output := []JellyMedia{}
	// Filter by time
	for _, media := range medias {
		switch media.Type {
		case "Movie":
			if media.DateCreated.After(timeSince) {
				output = append(output, media)
			}
		case "Series":
			if media.DateLastMediaAdded.After(timeSince) {
				output = append(output, media)
			}
		}
	}

	return output, nil
}

func (j *JellyUpdater) SendUpdateMessage(channelID string) {
	if j == nil || j.discordSession == nil {
		return
	}

	log.Printf("Posting to channel %s", channelID)

	MovieString := "**Movies**\n"
	TVString := "**TV Shows**\n"
	medias, err := j.GetRecentMediaSince(time.Now().Add(-1 * 24 * time.Hour)) // Daily
	if err != nil {
		log.Println("failed to get recent media")
		return
	}

	movieUpdates := 0
	tvUpdates := 0
	for _, media := range medias {
		switch media.Type {
		case "Movie":
			MovieString += media.Name + fmt.Sprintf(" (%d) ", media.ProductionYear) + "\n"
			movieUpdates++
		case "Series":
			TVString += media.Name + fmt.Sprintf(" (%d) ", media.ProductionYear) + fmt.Sprintf(" [New Episode Count: %d] ", media.ChildCount) + "\n"
			tvUpdates++
		}
	}

	if movieUpdates == 0 {
		MovieString = ""
	}
	if tvUpdates == 0 {
		TVString = ""
	}
	if movieUpdates == 0 && tvUpdates == 0 {
		// Nothing to do
		return
	}

	StringTemplate := fmt.Sprintf(`**New on Jellyfin Since Yesterday**
	
	%s
	%s
	`, MovieString, TVString)
	_, err = j.discordSession.ChannelMessageSend(channelID, StringTemplate)
	if err != nil {
		log.Println("err sending jellyfin update message", err)
	}
}

func (j *JellyUpdater) RecentHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if strings.ToLower(m.Content) != "!jellyfinrecent" {
		return
	}

	// Only certain channels can invoke this command
	keepGoing := false
	for _, allowedChannel := range JellyfinUpdateChannels {
		if m.ChannelID == allowedChannel {
			keepGoing = true
		}
	}
	if !keepGoing {
		return
	}

	j.SendUpdateMessage(m.ChannelID)
}

// Im not smart enough to understand why we need this closure
func (j *JellyUpdater) SendUpdateMessageToChannel(channelID string) func() {
	return func() { j.SendUpdateMessage(channelID) }
}
