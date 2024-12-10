package dota2matchreminder

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	cron "github.com/robfig/cron"
)

const doNotCookieMeGeneralChannel string = "1180043669009604700"
const doNotCookieMeIdiotBotChannel string = "1180047689044463666"

const myBestFriendsWebsite string = "https://dota.haglund.dev/v1/matches"

const (
	TeamNigma   string = "Nigma Galaxy"
	TeamFalcons string = "Team Falcons"
)

var trackedTeams []string = []string{TeamFalcons, TeamNigma}

var cachedMatches []Match = []Match{}

var reminderMap map[string]map[time.Time]string = map[string]map[time.Time]string{} // Map [ team name ] -> time as string to dedupe reminders (against opponent)

type Team struct {
	Name *string `json:"name"`
	URL  *string `json:"url"`
}

type Match struct {
	Hash       string  `json:"hash"`
	Teams      []*Team `json:"teams"`
	MatchType  *string `json:"matchType"`
	StartsAt   *string `json:"startsAt"`
	LeagueName *string `json:"leagueName"`
	LeagueUrl  *string `json:"leagueUrl"`
	StreamUrl  *string `json:"streamUrl"`
}

func StartDota2MatchReminder(disc *discordgo.Session) error {
	// Startup and then
	err := GetAndCacheMatchesAndSetUpReminders(disc)
	if err != nil {
		_, err := disc.ChannelMessageSend(doNotCookieMeIdiotBotChannel, err.Error())
		if err != nil {
			log.Println("Failed to send a message to discord in dota 2 reminder routine")
		}
	}
	// Start the cached updater to poll everyday at noon and midnight

	c := cron.New()

	err = c.AddFunc("0 0 * * *", func() {
		err := GetAndCacheMatchesAndSetUpReminders(disc)
		if err != nil {
			_, err := disc.ChannelMessageSend(doNotCookieMeIdiotBotChannel, err.Error())
			if err != nil {
				log.Println("Failed to send a message to discord in dota 2 reminder routine")
			}
		}
	})
	if err != nil {
		return err
	}

	err = c.AddFunc("0 12 * * *", func() {
		err := GetAndCacheMatchesAndSetUpReminders(disc)
		if err != nil {
			_, err := disc.ChannelMessageSend(doNotCookieMeIdiotBotChannel, err.Error())
			if err != nil {
				log.Println("Failed to send a message to discord in dota 2 reminder routine")
			}
		}
	})
	if err != nil {
		return err
	}

	c.Start()
	return nil
}

func GetAndCacheMatchesAndSetUpReminders(disc *discordgo.Session) error {
	resp, err := http.Get(myBestFriendsWebsite)
	if err != nil {
		return fmt.Errorf("failed to grab from dota 2 tournament api: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to grab from dota 2 tournament api: status %d", resp.StatusCode)
	}

	var matches []Match
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&matches); err != nil {
		return fmt.Errorf("failed to deserialize from dota 2 tournament api: %v", err)
	}

	cachedMatches = matches
	for _, team := range trackedTeams {
		CheckMatchesForTeamAndCreateReminderTimers(disc, team)
	}

	return nil
}

func CheckMatchesForTeamAndCreateReminderTimers(disc *discordgo.Session, team string) {
	for _, match := range cachedMatches {
		if len(match.Teams) != 2 {
			// What is this match
			continue
		}
		if match.Teams[0].Name == nil || match.Teams[1].Name == nil {
			// What is this match
			continue
		}
		if *match.Teams[0].Name != team && *match.Teams[1].Name != team {
			// This match doesnt matter
			continue
		}

		if match.StartsAt == nil {
			continue
		}
		// Parse time
		matchTime, err := time.Parse(time.RFC3339, *match.StartsAt)
		if err != nil {
			log.Printf("failed to parse match time from %+v \n", *match.StartsAt)
		}

		var opponent string
		if *match.Teams[0].Name == team {
			opponent = *match.Teams[1].Name
		} else {
			opponent = *match.Teams[0].Name
		}

		_, ok := reminderMap[team]
		if !ok {
			reminderMap[team] = map[time.Time]string{}
		}

		_, ok = reminderMap[team][matchTime]
		if ok {
			continue
		} else {
			reminderMap[team][matchTime] = opponent
		}

		if time.Now().Before(matchTime.Add(-30 * time.Minute)) {
			time.AfterFunc(time.Until(matchTime.Add(-30*time.Minute)), ClosureForMatchSend(match, disc))
		}
	}
}

func ClosureForMatchSend(match Match, disc *discordgo.Session) func() {
	return func() {
		content := fmt.Sprintf(
			`Dota 2 Tournament Match in 30 minutes: %s
				**%s vs %s**`, *match.LeagueName, *match.Teams[0].Name, *match.Teams[1].Name)
		_, err := disc.ChannelMessageSend(doNotCookieMeGeneralChannel, content)
		if err != nil {
			log.Printf("failed to send discord message %+v \n", err)
		}
	}
}

// Handlers
func HandleDota2Matches(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	// for sorting
	type opponentTime struct {
		opponent  string
		matchTime time.Time
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "!dota2matches") {
		fixedPST := time.FixedZone("PST", -8*3600)
		content := fmt.Sprintf("Upcoming Dota 2 Promatches for %v \n", trackedTeams)

		for team, matchTimeMap := range reminderMap {
			content += "\n"
			content += fmt.Sprintf("**%s** is playing: \n", team)
			sortable := []opponentTime{}
			for matchTime, opponent := range matchTimeMap {
				sortable = append(sortable, opponentTime{
					opponent:  opponent,
					matchTime: matchTime,
				})
			}

			// Sort by time
			sort.Slice(sortable, func(i int, j int) bool {
				return sortable[i].matchTime.Before(sortable[j].matchTime)
			})

			for _, oppTime := range sortable {
				if time.Now().Add(-2*time.Hour).Before(oppTime.matchTime) && time.Now().After(oppTime.matchTime) {
					content += fmt.Sprintf("[**Possibly Live**] against **%s** at %s (%s ago) \n", oppTime.opponent, oppTime.matchTime.In(fixedPST).Format(time.RFC1123), fmtDuration(time.Until(oppTime.matchTime)))
				}
				if time.Now().Before(oppTime.matchTime) {
					content += fmt.Sprintf("against **%s** at %s (in %s) \n", oppTime.opponent, oppTime.matchTime.In(fixedPST).Format(time.RFC1123), fmtDuration(time.Until(oppTime.matchTime)))
				}
			}
		}
		s.ChannelMessageSend(m.ChannelID, content)
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02dh %02dm", h, m)
}
