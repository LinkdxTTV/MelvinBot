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

	"MelvinBot/src/dota2matchreminder"
	"MelvinBot/src/jellyfin"
	"MelvinBot/src/nisha"
	"MelvinBot/src/nlquotes"
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
		c.AddFunc("0 0 4 * * *", jf.SendUpdateMessageToChannel(channel))
	}

	c.Start()

	err = dota2matchreminder.StartDota2MatchReminder(bot.discord)
	if err != nil {
		log.Println("error in dota 2 match reminder", err)
	}

	// Add message handlers here
	bot.discord.AddHandler(goMessageHandler(monkaS))
	bot.discord.AddHandler(goMessageHandler(stats.TrackStats))
	bot.discord.AddHandler(goMessageHandler(stats.PrintStats))
	bot.discord.AddHandler(goMessageHandler(nisha.DidSomebodySaySex))
	bot.discord.AddHandler(goMessageHandler(nisha.ThisIsNotADvd))
	bot.discord.AddHandler(goMessageHandler(nisha.GeorgeCarlin))
	bot.discord.AddHandler(goMessageHandler(nisha.Tetazoo))
	bot.discord.AddHandler(goMessageHandler(nisha.Glounge))
	bot.discord.AddHandler(goMessageHandler(nisha.Iiwii))
	bot.discord.AddHandler(goMessageHandler(nisha.Lethimcook))
	bot.discord.AddHandler(goMessageHandler(nisha.Miami))
	bot.discord.AddHandler(goMessageHandler(nisha.KillDamian))
	bot.discord.AddHandler(goMessageHandler(quotes.HandleQuote))
	bot.discord.AddHandler(goMessageHandler(quotes.RemoveQuote))
	bot.discord.AddHandler(goMessageHandler(nlquotes.HandleNLQuote))
	bot.discord.AddHandler(goMessageHandler(jf.RecentHandler))
	bot.discord.AddHandler(goMessageHandler(dota2matchreminder.HandleDota2Matches))

	// add other handlers here (reacts etc)
	bot.discord.AddHandler(goReactionAddHandler(quotes.AddQuote))
	bot.discord.AddHandler(goReactionAddHandler(pinFromReaction))
	bot.discord.AddHandler(goReactionRemoveHandler(unpinFromReaction))

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

func goMessageHandler(f func(*disc.Session, *disc.MessageCreate)) func(*disc.Session, *disc.MessageCreate) {
	return func(s *disc.Session, mc *disc.MessageCreate) {
		go f(s, mc)
	}
}

func goReactionAddHandler(f func(*disc.Session, *disc.MessageReactionAdd)) func(*disc.Session, *disc.MessageReactionAdd) {
	return func(s *disc.Session, mc *disc.MessageReactionAdd) {
		go f(s, mc)
	}
}

func goReactionRemoveHandler(f func(*disc.Session, *disc.MessageReactionRemove)) func(*disc.Session, *disc.MessageReactionRemove) {
	return func(s *disc.Session, mc *disc.MessageReactionRemove) {
		go f(s, mc)
	}
}
