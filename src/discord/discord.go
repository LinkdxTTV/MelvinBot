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
	"MelvinBot/src/stats"
	"MelvinBot/src/store"
	"MelvinBot/src/util"

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
	bot.discord.AddHandler(didSomebodySaySex)
	bot.discord.AddHandler(thisIsNotADvd)
	bot.discord.AddHandler(georgeCarlin)
	bot.discord.AddHandler(tetazoo)
	bot.discord.AddHandler(glounge)
	bot.discord.AddHandler(iiwii)
	bot.discord.AddHandler(lethimcook)
	bot.discord.AddHandler(randomQuote)
	bot.discord.AddHandler(miami)

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

func didSomebodySaySex(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "sex") {
		s.ChannelMessageSend(m.ChannelID, "did somebody say sex???")
	}
}

func thisIsNotADvd(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if m.Content != "!stop" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "STOP! STOP! STOP! This is NOT a DVD. This is NOT A DVD. THIS IS NOT A DVD. This is a BACKER CARD. It's a CARD for COLLECTORS. This is a MOVIE CARD. THIS IS NOT A DVD. STOP! READ. READ THE DESCRIPTION.")

}

func georgeCarlin(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if m.Content != "!rsbs" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "RATSHIT BATSHIT DIRTY OLD TWAT 69 ASSHOLES TIED IN A KNOT HOORAY LIZARD SHIT FUCK")
}

func tetazoo(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "tetazoo") {
		s.ChannelMessageSend(m.ChannelID, "TETAZOO IS NOT A HIVEMIND")
	}
}

func glounge(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "where are you") {
		s.ChannelMessageSend(m.ChannelID, "update tetazoo glounge")
	}
}

func iiwii(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.Content != "!iiwii" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "it EEEEEEES what it eees")
}

func lethimcook(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "cook") {
		s.ChannelMessageSend(m.ChannelID, "https://i.kym-cdn.com/entries/icons/original/000/041/943/1aa1blank.png")
	}
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

	allQuotes, _ := parse.ParseAndDedupCsv()
	s.ChannelMessageSend(m.ChannelID, allQuotes[rand.Intn(len(allQuotes))])
}

func miami(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_id {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "miami") || strings.Contains(strings.ToLower(m.Message.Content), "spring break") {
		year := []rune(fmt.Sprint(time.Now().Year()))
		year[1] := 'k'
		s.ChannelMessageSend(m.ChannelID, fmt.Sprint("SPRING BREAK MIAMI %v WOOOOOOOO"), string(year))
	}
}
