package discord

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"MelvinBot/src/jellyfin"
	"MelvinBot/src/nisha"
	"MelvinBot/src/quotes"
	"MelvinBot/src/stats"
	"MelvinBot/src/store"
	"MelvinBot/src/util"

	disc "github.com/bwmarrin/discordgo"
	cron "github.com/robfig/cron"
)

type Bot struct {
	discord   *disc.Session
	store     store.Storage
	quotes    store.Storage
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

	quotes, err := store.NewLocalStorage(&quotes.GuildIDToQuoteDatabase, quotes.Filepath)
	if err != nil {
		log.Fatal("could not get quotes")
	}

	return Bot{discord, storage, quotes, statsFile}
}

func (bot Bot) RunBot() {

	// Init stats
	if _, err := os.Stat(bot.statsfile); errors.Is(err, os.ErrNotExist) {
		bot.store.Put()
	}

	err := bot.store.Get()
	if err != nil {
		log.Fatal(err)
	}

	bot.store.SyncOnTimer(1 * time.Minute)

	// Init quotes
	if _, err := os.Stat(quotes.Filepath); errors.Is(err, os.ErrNotExist) {
		bot.quotes.Put()
	}

	err = bot.quotes.Get()
	if err != nil {
		log.Fatal(err)
	}

	bot.quotes.SyncOnTimer(1 * time.Minute)
	jf := jellyfin.NewJellyUpdater(bot.discord)
	// For scheduled jobs
	c := cron.New()
	// Send quote at 8:00AM every day
	quoteBoardChannelID := "1093406748783693854"
	c.AddFunc("0 0 8 * * *", func() { sendRandomQuote(bot.discord, quoteBoardChannelID, util.Wolfcord_GuildID) }) // Magic bullshit that puts it at midnight PST
	for _, channel := range jellyfin.JellyfinUpdateChannels {
		c.AddFunc("0 0 1 * * *", func() { jf.SendUpdateMessageToChannel(channel) })
	}

	c.Start()

	// Add handlers here
	bot.discord.AddHandler(monkaS)
	bot.discord.AddHandler(stats.TrackStats)
	bot.discord.AddHandler(stats.PrintStats)
	bot.discord.AddHandler(pinFromReaction)
	bot.discord.AddHandler(unpinFromReaction)
	bot.discord.AddHandler(nisha.DidSomebodySaySex)
	bot.discord.AddHandler(nisha.ThisIsNotADvd)
	bot.discord.AddHandler(nisha.GeorgeCarlin)
	bot.discord.AddHandler(nisha.Tetazoo)
	bot.discord.AddHandler(nisha.Glounge)
	bot.discord.AddHandler(nisha.Iiwii)
	bot.discord.AddHandler(nisha.Lethimcook)
	bot.discord.AddHandler(nisha.Miami)
	bot.discord.AddHandler(quotes.HandleQuote)
	bot.discord.AddHandler(quotes.AddQuote)
	bot.discord.AddHandler(quotes.RemoveQuote)
	bot.discord.AddHandler(jf.RecentHandler)

	err = bot.discord.Open()
	if err != nil {
		log.Fatal("couldnt open connection", err)
	}
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Place stats one last time for consistency
	err = bot.store.Put()
	if err != nil {
		log.Printf("failed put call on shutdown: %v", err)
	}

	err = bot.quotes.Put()
	if err != nil {
		log.Printf("failed put call on shutdown: %v", err)
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

func sendRandomQuote(s *disc.Session, channelID string, guildID string) {
	database, ok := quotes.GuildIDToQuoteDatabase[guildID]
	// Just in case we never have init'd quotes in this server
	if !ok {
		newDatabase := &quotes.QuoteDatabase{
			Quotes:                      []quotes.Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			Lock:                        &sync.Mutex{},
		}
		quotes.GuildIDToQuoteDatabase[guildID] = newDatabase
		database = newDatabase
	}
	database.Lock.Lock()
	defer database.Lock.Unlock()

	totalQuotes := len(database.Quotes)
	if totalQuotes == 0 {
		return
	}

	database.SendQuote(s, channelID, rand.Intn(totalQuotes), totalQuotes)
}
