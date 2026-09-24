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

const jellyfinBaseURL = "http://localhost:8096/jelly"

type QueryResult struct {
	Items []JellyMedia `json:"Items"`
}

func (j *JellyUpdater) makeGetRequest(endpoint string) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Println("failed to parse url", endpoint, err)
		return nil, err
	}

	authHeader := []string{fmt.Sprintf("MediaBrowser Client=\"Jellyfin Web\", Device=\"Firefox\", DeviceId=\"abcdefg\", Version=\"10.7.6\", Token=\"%s\"", j.apiKey)}

	client := http.Client{}
	resp, err := client.Do(&http.Request{
		Method: http.MethodGet,
		URL:    u,
		Header: http.Header{
			"Authorization": authHeader,
		},
	})
	if err != nil {
		log.Println("failed to make http req", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("failed to make http req, got invalid status code", resp.StatusCode, "for url", endpoint)
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("could not read response body", err)
		return nil, err
	}
	return b, nil
}

func (j *JellyUpdater) GetFolderIDs() (map[string]string, error) {
	endpoint := fmt.Sprintf("%s/UserViews?userId=%s", jellyfinBaseURL, j.userID)
	b, err := j.makeGetRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var res struct {
		Items []struct {
			Name string `json:"Name"`
			Id   string `json:"Id"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	folderMap := make(map[string]string)
	for _, item := range res.Items {
		folderMap[strings.ToLower(item.Name)] = item.Id
	}
	return folderMap, nil
}

func (j *JellyUpdater) GetRecentMovies(folderID string, timeSince time.Time) ([]JellyMedia, error) {
	endpoint := fmt.Sprintf("%s/Items?userId=%s&parentId=%s&recursive=true&includeItemTypes=Movie&sortBy=DateCreated&sortOrder=Descending&fields=DateCreated,ProductionYear&limit=50", jellyfinBaseURL, j.userID, folderID)
	b, err := j.makeGetRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var res QueryResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	var movies []JellyMedia
	for _, item := range res.Items {
		if item.DateCreated.After(timeSince) {
			movies = append(movies, item)
		}
	}
	return movies, nil
}

func (j *JellyUpdater) GetRecentShows(folderID string, timeSince time.Time) ([]TVEpisodes, error) {
	endpoint := fmt.Sprintf("%s/Items?userId=%s&parentId=%s&includeItemTypes=Series&sortBy=DateLastContentAdded,DateCreated&sortOrder=Descending&fields=DateCreated,DateLastMediaAdded,ProductionYear&limit=50", jellyfinBaseURL, j.userID, folderID)
	b, err := j.makeGetRequest(endpoint)
	if err != nil {
		return nil, err
	}

	var res QueryResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}

	var tvshows []TVEpisodes
	for _, item := range res.Items {
		episodes := j.GetTVSeriesWithEpisodes(item.Name, item.Id, timeSince)
		if len(episodes.Items) > 0 {
			tvshows = append(tvshows, episodes)
		}
	}
	return tvshows, nil
}

func (j *JellyUpdater) GetRecentAnime(folderID string, timeSince time.Time) ([]TVEpisodes, []JellyMedia, error) {
	endpoint := fmt.Sprintf("%s/Items?userId=%s&parentId=%s&recursive=true&includeItemTypes=Series,Movie&sortBy=DateLastContentAdded,DateCreated&sortOrder=Descending&fields=DateCreated,DateLastMediaAdded,ProductionYear&limit=50", jellyfinBaseURL, j.userID, folderID)
	b, err := j.makeGetRequest(endpoint)
	if err != nil {
		return nil, nil, err
	}

	var res QueryResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, nil, err
	}

	var animeShows []TVEpisodes
	var animeMovies []JellyMedia
	for _, item := range res.Items {
		switch item.Type {
		case "Movie":
			if item.DateCreated.After(timeSince) {
				animeMovies = append(animeMovies, item)
			}
		case "Series":
			episodes := j.GetTVSeriesWithEpisodes(item.Name, item.Id, timeSince)
			if len(episodes.Items) > 0 {
				animeShows = append(animeShows, episodes)
			}
		}
	}
	return animeShows, animeMovies, nil
}

func (j *JellyUpdater) GetRecentMediaSince(timeSince time.Time) (movies []JellyMedia, tvshows []TVEpisodes, animeShows []TVEpisodes, animeMovies []JellyMedia, err error) {
	folderIDs, err := j.GetFolderIDs()
	if err != nil {
		log.Println("failed to get jellyfin folder IDs:", err)
		folderIDs = make(map[string]string)
	}

	movieFolderID := folderIDs["movies"]
	if movieFolderID == "" {
		movieFolderID = folderIDs["movie"]
	}

	showFolderID := folderIDs["shows"]
	if showFolderID == "" {
		showFolderID = folderIDs["tv shows"]
	}
	if showFolderID == "" {
		showFolderID = folderIDs["tv"]
	}

	animeFolderID := folderIDs["anime"]

	if movieFolderID != "" {
		m, err := j.GetRecentMovies(movieFolderID, timeSince)
		if err != nil {
			log.Println("failed to get recent movies:", err)
		} else {
			movies = m
		}
	} else {
		log.Println("could not find folder ID for Movies")
	}

	if showFolderID != "" {
		s, err := j.GetRecentShows(showFolderID, timeSince)
		if err != nil {
			log.Println("failed to get recent shows:", err)
		} else {
			tvshows = s
		}
	} else {
		log.Println("could not find folder ID for Shows")
	}

	if animeFolderID != "" {
		as, am, err := j.GetRecentAnime(animeFolderID, timeSince)
		if err != nil {
			log.Println("failed to get recent anime:", err)
		} else {
			animeShows = as
			animeMovies = am
		}
	} else {
		log.Println("could not find folder ID for Anime")
	}

	return movies, tvshows, animeShows, animeMovies, nil
}

func (j *JellyUpdater) SendUpdateMessage(channelID string) {
	if j == nil || j.discordSession == nil {
		return
	}

	log.Printf("Posting to channel %s", channelID)

	movies, tvshows, animeShows, animeMovies, err := j.GetRecentMediaSince(time.Now().Add(-1 * 24 * time.Hour)) // Daily
	if err != nil {
		log.Println("failed to get recent media:", err)
		return
	}

	MovieString := ""
	if len(movies) > 0 {
		MovieString = "**Movies**\n"
		for _, media := range movies {
			MovieString += media.Name + fmt.Sprintf(" (%d) ", media.ProductionYear) + "\n"
		}
	}

	TVString := ""
	if len(tvshows) > 0 {
		TVString = "**TV Shows**\n"
		for _, show := range tvshows {
			TVString += show.Series + fmt.Sprintf(" (%d)", show.Items[0].ProductionYear) + fmt.Sprintf(" [ %d New Episode(s) ] ", len(show.Items)) + "\n"
		}
	}

	AnimeString := ""
	if len(animeShows) > 0 || len(animeMovies) > 0 {
		AnimeString = "**Anime**\n"
		for _, media := range animeMovies {
			AnimeString += media.Name + fmt.Sprintf(" (%d) ", media.ProductionYear) + "\n"
		}
		for _, show := range animeShows {
			AnimeString += show.Series + fmt.Sprintf(" (%d)", show.Items[0].ProductionYear) + fmt.Sprintf(" [ %d New Episode(s) ] ", len(show.Items)) + "\n"
		}
	}

	if MovieString == "" && TVString == "" && AnimeString == "" {
		// Nothing to do
		_, err = j.discordSession.ChannelMessageSend(channelID, "No new items in last 24 hours")
		if err != nil {
			log.Println("err sending jellyfin update message", err)
		}
		return
	}

	var sections []string
	if MovieString != "" {
		sections = append(sections, MovieString)
	}
	if TVString != "" {
		sections = append(sections, TVString)
	}
	if AnimeString != "" {
		sections = append(sections, AnimeString)
	}

	StringTemplate := fmt.Sprintf("**New on Jellyfin Since Yesterday**\n\n%s", strings.Join(sections, "\n"))
	_, err = j.discordSession.ChannelMessageSend(channelID, StringTemplate)
	if err != nil {
		log.Println("err sending jellyfin update message", err)
	}
}

func (j *JellyUpdater) GetTVSeriesWithEpisodes(seriesName string, seriesID string, since time.Time) TVEpisodes {
	EpisodesEndpoint := fmt.Sprintf("%s/Shows/%s/Episodes?fields=DateCreated", jellyfinBaseURL, seriesID)
	b, err := j.makeGetRequest(EpisodesEndpoint)
	if err != nil {
		log.Println("err getting episodes for series id", seriesID, err)
		return TVEpisodes{}
	}

	seriesAndEpisodes := TVEpisodes{Series: seriesName}
	err = json.Unmarshal(b, &seriesAndEpisodes)
	if err != nil {
		log.Println("err unmarshaling episodes", err)
		return TVEpisodes{}
	}

	// Filter by time
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
