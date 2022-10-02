package discord

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"MelvinBot/src/stats"
	"MelvinBot/src/store"

	disc "github.com/bwmarrin/discordgo"
)

type Bot struct {
	discord   *disc.Session
	store     store.Storage
	statsfile string
}

func NewBot(token string) Bot {
	discord, err := disc.New("Bot " + token)
	if err != nil {
		log.Fatal("could not connect to discord")
	}

	statsFile := "/etc/melvinstats"
	storage, err := store.NewLocalStorage(statsFile)
	if err != nil {
		log.Fatal("could not get local stats")
	}

	return Bot{discord, storage, statsFile}
}

func (bot Bot) RunBot() {

	if _, err := os.Stat(bot.statsfile); errors.Is(err, os.ErrNotExist) {
		bot.store.PutStats()
	}

	err := bot.store.GetStats()
	if err != nil {
		log.Fatal(err)
	}

	bot.store.SyncStatsOnTimer(1 * time.Minute)

	// Add handlers here
	bot.discord.AddHandler(monkaS)
	bot.discord.AddHandler(stats.TrackStats)
	bot.discord.AddHandler(stats.PrintStats)
	bot.discord.AddHandler(pinFromReaction)
	bot.discord.AddHandler(unpinFromReaction)

	err = bot.discord.Open()
	if err != nil {
		log.Fatal("couldnt open connection", err)
	}
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Place stats one last time for consistency
	err = bot.store.PutStats()
	if err != nil {
		log.Printf("failed dynamo put call on shutdown: %v", err)
	}
	// Cleanly close down the Discord session.
	bot.discord.Close()
}

// Handlers
func monkaS(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "monkas") {
		s.ChannelMessageSend(m.ChannelID, "monkaS")
	}
}
