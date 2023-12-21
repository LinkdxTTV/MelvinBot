package quotes

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	disc "github.com/bwmarrin/discordgo"
)

const Filepath string = "/home/nelly/apps/bot/melvinquotes"

type QuoteDatabase struct {
	Quotes                      []Quote
	MapFromAuthorToQuoteIndices map[string][]int
	Lock                        *sync.Mutex
}

type Quote struct {
	Quote  string
	Author string
	UserID string
}

func (q *Quote) String() string {
	return fmt.Sprintf("%s -@%s", q.Quote, q.Author)
}

var GuildIDToQuoteDatabase = map[string]*QuoteDatabase{}

func AddQuote(s *disc.Session, m *disc.MessageReactionAdd) {
	if m.MessageReaction.Emoji.Name != "💬" {
		return
	}

	// Dedupe quotes
	msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		return
	}
	numQuotes := 0
	for _, reaction := range msg.Reactions {
		if reaction.Emoji.Name == "💬" {
			numQuotes++
			if numQuotes > 1 {
				// Duplicate
				return
			}
		}
	}

	guildID := m.GuildID
	message, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		log.Printf("error setting quote, could not get message for %s, %s: %v", m.ChannelID, m.MessageID, err)
		return
	}

	// Just in case we have never made a quote for this guild?
	database, ok := GuildIDToQuoteDatabase[guildID]
	if !ok {
		newDatabase := &QuoteDatabase{
			Quotes:                      []Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[guildID] = newDatabase
		database = newDatabase
	}
	database.Lock.Lock()
	defer database.Lock.Unlock()

	database.Quotes = append(database.Quotes, Quote{
		Quote:  message.Content,
		Author: message.Author.Username,
		UserID: message.Author.ID,
	})

	// Save by username as well
	_, ok = database.MapFromAuthorToQuoteIndices[message.Author.Username]
	if !ok {
		database.MapFromAuthorToQuoteIndices[message.Author.Username] = []int{}

		database.MapFromAuthorToQuoteIndices[message.Author.Username] = append(database.MapFromAuthorToQuoteIndices[message.Author.Username], len(database.Quotes)-1)

		// Finally ack
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("New quote added at number %d", len(database.Quotes)-1))
	}
}

// TODO
func RemoveQuote(s *disc.Session, m *disc.MessageReactionAdd) {
	if m.MessageReaction.Emoji.Name != "💬" {
		return
	}

	msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		return
	}
	for _, reaction := range msg.Reactions {
		if reaction.Emoji.Name == "💬" {
			return
		}
	}

	database, ok := GuildIDToQuoteDatabase[m.GuildID]
	if !ok {
		newDatabase := &QuoteDatabase{
			Quotes:                      []Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[m.GuildID] = newDatabase
		database = newDatabase
	}
	database.Lock.Lock()
	defer database.Lock.Unlock()

	// TODO
}

func HandleQuote(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if !strings.Contains(m.Content, "!quote") {
		return
	}

	guildID := m.GuildID

	database, ok := GuildIDToQuoteDatabase[guildID]
	// Just in case we never have init'd quotes in this server
	if !ok {
		newDatabase := &QuoteDatabase{
			Quotes:                      []Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[guildID] = newDatabase
		database = newDatabase
	}

	database.Lock.Lock()
	defer database.Lock.Unlock()

	totalQuotes := len(database.Quotes)
	if totalQuotes == 0 {
		_, err := s.ChannelMessageSend(m.ChannelID, "This server has no saved quotes yet!")
		if err != nil {
			log.Printf("error sending message for random quote %v", err)
		}
		return
	}

	// Random quote
	if m.Content == "!quote" {
		database.sendQuote(s, m, rand.Intn(totalQuotes), totalQuotes)
		return
	}

	split := strings.Split(m.Content, " ")
	if len(split) != 2 {
		// Ok they put in some true garbage
		s.ChannelMessageSend(m.ChannelID, "You must specify a quote id (its a number) like !quote 5")
		return
	}
	quoteInt, err := strconv.Atoi(split[1])
	if err != nil {
		// They put in more garbage
		s.ChannelMessageSend(m.ChannelID, "You must specify a quote id (its a number) like !quote 5")
		return
	}
	database.sendQuote(s, m, quoteInt, totalQuotes)
}

func (d *QuoteDatabase) sendQuote(s *disc.Session, m *disc.MessageCreate, index int, totalQuotes int) {

	if index >= totalQuotes {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Sorry we only have up to quote %d", totalQuotes-1))
		return
	}

	// Check if we can mention this author
	quote := d.Quotes[index]
	author := quote.Author
	user, err := s.User(quote.UserID)
	if err == nil && user != nil {
		author = user.Mention()
	}

	_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Quote #%d: %s -%s", index, quote.Quote, author))
	if err != nil {
		log.Printf("error sending message for random quote %v", err)
	}
}
