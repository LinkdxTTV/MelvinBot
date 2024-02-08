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
	Id                 string
}

type TVEpisodes struct {
	Items  []JellyMedia
	Series string
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
func (j *JellyUpdater) GetRecentMediaSince(timeSince time.Time) ([]JellyMedia, []TVEpisodes, error) {
	recentMediaEndpoint := fmt.Sprintf("http://localhost:8096/jelly/Users/%s/Items/Latest?fields=DateLastMediaAdded,DateCreated&enableImages=false&enableUserData=false", j.userID)

	url, err := url.Parse(recentMediaEndpoint)
	if err != nil {
		log.Println("failed to parse jellyfin recent media url", err)
		return nil, nil, err
	}

	var AuthorizationHeader []string = []string{fmt.Sprintf("MediaBrowser Client=\"Jellyfin Web\", Device=\"Firefox\", DeviceId=\"abcdefg\", Version=\"10.7.6\", Token=\"%s\"", j.apiKey)}

	client := http.Client{}
	resp, err := client.Do(
		&http.Request{
			Method: http.MethodGet,
			URL:    url,
			Header: http.Header{
				"Authorization": AuthorizationHeader,
			},
		},
	)
	if err != nil {
		log.Println("failed to make http req", err)
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Println("failed to make http req, got invalid status code", resp.StatusCode)
		return nil, nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("could not read response body", err)
		return nil, nil, err
	}

	medias := []JellyMedia{}
	err = json.Unmarshal(b, &medias)
	if err != nil {
		return nil, nil, err
	}

	movies := []JellyMedia{}
	tvshows := []TVEpisodes{}
	// Filter by time
	for _, media := range medias {
		switch media.Type {
		case "Movie":
			if media.DateCreated.After(timeSince) {
				movies = append(movies, media)
			}
		case "Series":
			tvshows = append(tvshows, j.GetTVSeriesWithEpisodes(media.Name, media.Id, timeSince))
		}
	}

	return movies, tvshows, nil
}

func (j *JellyUpdater) SendUpdateMessage(channelID string) {
	if j == nil || j.discordSession == nil {
		return
	}

	log.Printf("Posting to channel %s", channelID)

	MovieString := "**Movies**\n"
	TVString := "**TV Shows**\n"
	movies, tvshows, err := j.GetRecentMediaSince(time.Now().Add(-1 * 24 * time.Hour)) // Daily
	if err != nil {
		log.Println("failed to get recent media")
		return
	}

	movieUpdates := 0
	tvUpdates := 0
	for _, media := range movies {
		MovieString += media.Name + fmt.Sprintf(" (%d) ", media.ProductionYear) + "\n"
		movieUpdates++
	}

	for _, show := range tvshows {
		if len(show.Items) != 0 {
			TVString += show.Series + fmt.Sprintf(" (%d)", show.Items[0].ProductionYear) + fmt.Sprintf(" [ %d New Episode(s) ] ", len(show.Items)) + "\n"
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

func (j *JellyUpdater) GetTVSeriesWithEpisodes(seriesName string, seriesID string, since time.Time) TVEpisodes {
	EpisodesEndpoint := fmt.Sprintf("http://localhost:8096/jelly/Shows/%s/Episodes?fields=DateCreated", seriesID)
	url, err := url.Parse(EpisodesEndpoint)
	if err != nil {
		log.Println("couldnt parse url", EpisodesEndpoint)
		return TVEpisodes{}
	}

	var AuthorizationHeader []string = []string{fmt.Sprintf("MediaBrowser Client=\"Jellyfin Web\", Device=\"Firefox\", DeviceId=\"abcdefg\", Version=\"10.7.6\", Token=\"%s\"", j.apiKey)}

	client := &http.Client{}
	resp, err := client.Do(&http.Request{
		Method: http.MethodGet,
		URL:    url,
		Header: http.Header{
			"Authorization": AuthorizationHeader,
		},
	})

	if err != nil {
		log.Println("err getting episodes", err)
		return TVEpisodes{}
	}
	if resp.StatusCode != http.StatusOK {
		log.Println("err with request for episodes of series id", seriesID)
		return TVEpisodes{}
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("err reading resp when grabbing episodes", err)
		return TVEpisodes{}
	}
	seriesAndEpisodes := TVEpisodes{Series: seriesName}

	err = json.Unmarshal(b, &seriesAndEpisodes)
	if err != nil {
		log.Println("err unmarshaling episodes", err)
		return TVEpisodes{}
	}

	// Lets filter at this point
	temp := []JellyMedia{}
	for _, item := range seriesAndEpisodes.Items {
		if item.DateCreated.After(since) {
			temp = append(temp, item)
		}
	}
	seriesAndEpisodes.Items = temp

	return seriesAndEpisodes
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
