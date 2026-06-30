package stats

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	disc "github.com/bwmarrin/discordgo"
)

type Stats struct {
	StatMap map[string]int
	Lock    *sync.Mutex
}

var StatsPerGuild map[string]*Stats = map[string]*Stats{}

func TrackStats(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	guildStats, ok := StatsPerGuild[m.GuildID]
	if !ok {
		guildStats = &Stats{
			StatMap: map[string]int{},
			Lock:    &sync.Mutex{},
		}
		// What are the chances this races
		StatsPerGuild[m.GuildID] = guildStats
	}

	guildStats.Lock.Lock()
	defer guildStats.Lock.Unlock()

	_, ok = guildStats.StatMap[m.Author.Username]
	if ok {
		guildStats.StatMap[m.Author.Username]++
	} else {
		guildStats.StatMap[m.Author.Username] = 1
	}
}

func PrintStats(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.Content != "!stats" {
		return
	}

	guildStats, ok := StatsPerGuild[m.GuildID]
	if !ok {
		s.ChannelMessageSend(m.ChannelID, "Sorry I'm not tracking stats for this server")
		return
	}

	guildStats.Lock.Lock()

	// Sort and create stats array
	type MelvinPosts struct {
		name  string
		posts int
	}

	sortable := []MelvinPosts{}
	for username, posts := range guildStats.StatMap {

		sortable = append(sortable, MelvinPosts{name: username, posts: posts})
	}
	// Don't need lock anymore
	guildStats.Lock.Unlock()

	sort.Slice(sortable, func(i, j int) bool {
		return sortable[i].posts > sortable[j].posts
	})

	var statsMessage strings.Builder
	statsMessage.WriteString("Melvin Posts Leaderboard:")
	for _, message := range sortable {
		statsMessage.WriteString(fmt.Sprintf("\n%s : %d", message.name, message.posts))
	}

	s.ChannelMessageSend(m.ChannelID, statsMessage.String())
}
