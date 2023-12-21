package discord

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	parse "MelvinBot/src/csv"
	"MelvinBot/src/nisha"
	"MelvinBot/src/stats"
	"MelvinBot/src/store"
	"MelvinBot/src/util"

	disc "github.com/bwmarrin/discordgo"
	cron "github.com/robfig/cron"
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
	storage, err := store.NewLocalStorage(&stats.StatsPerGuild, statsFile)
	if err != nil {
		log.Fatal("could not get local stats")
	}

	return Bot{discord, storage, statsFile}
}

func (bot Bot) RunBot() {

	if _, err := os.Stat(bot.statsfile); errors.Is(err, os.ErrNotExist) {
		bot.store.Put()
	}

	err := bot.store.Get()
	if err != nil {
		log.Fatal(err)
	}

	bot.store.SyncOnTimer(1 * time.Minute)

	// For scheduled jobs
	c := cron.New()
	// Send quote at 8:00AM every day
	quoteBoardChannelID := "1093406748783693854"
	c.AddFunc("0 0 8 * * *", func() { sendRandomQuote(bot.discord, quoteBoardChannelID) })
	c.Start()

	// Add handlers here
	bot.discord.AddHandler(monkaS)
	bot.discord.AddHandler(stats.TrackStats)
	bot.discord.AddHandler(stats.PrintStats)
	bot.discord.AddHandler(pinFromReaction)
	bot.discord.AddHandler(unpinFromReaction)
	bot.discord.AddHandler(randomQuote)
	bot.discord.AddHandler(nisha.DidSomebodySaySex)
	bot.discord.AddHandler(nisha.ThisIsNotADvd)
	bot.discord.AddHandler(nisha.GeorgeCarlin)
	bot.discord.AddHandler(nisha.Tetazoo)
	bot.discord.AddHandler(nisha.Glounge)
	bot.discord.AddHandler(nisha.Iiwii)
	bot.discord.AddHandler(nisha.Lethimcook)
	bot.discord.AddHandler(nisha.Miami)

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
	err = bot.store.Put()
	if err != nil {
		log.Printf("failed dynamo put call on shutdown: %v", err)
	}

	c.Stop()

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

func sendRandomQuote(s *disc.Session, channelID string) {
	allQuotes, _ := parse.ParseAndDedupCsv()
	s.ChannelMessageSend(channelID, allQuotes[rand.Intn(len(allQuotes))])
}

func randomQuote(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if m.Content != "!quote" {
		return
	}

	sendRandomQuote(s, m.ChannelID)
}
