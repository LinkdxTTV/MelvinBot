package quotes

import (
	"MelvinBot/src/util"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	disc "github.com/bwmarrin/discordgo"
)

const Filepath string = "/home/nelly/apps/bot/melvinquotes"
const DeletedQuoteString = "This quote has been deleted"

type QuoteDatabase struct {
	Quotes                      []Quote
	MapFromAuthorToQuoteIndices map[string][]int
	QuoteGraveyard              []int // The quote graveyard is a list of indexes where we have deleted quotes but do not want to reorder the array
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

	newQuoteID := AddQuoteToDatabase(guildID, message.Content, message.Author.Username, message.Author.ID)
	// Finally ack
	err = util.SendSelfDestructingMessage(s, m.ChannelID, fmt.Sprintf("Added quote [#%d]: ```%s``` -%s", newQuoteID, message.Content, message.Author.Username), 10*time.Second)
	if err != nil {
		log.Printf("err sending self destructing msg: %v", err)
	}
}

func AddQuoteToDatabase(guildID string, quote string, author string, userID string) int {
	// Just in case we have never made a quote for this guild?
	database, ok := GuildIDToQuoteDatabase[guildID]
	if !ok {
		newDatabase := &QuoteDatabase{
			Quotes:                      []Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			QuoteGraveyard:              []int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[guildID] = newDatabase
		database = newDatabase
	}
	database.Lock.Lock()
	defer database.Lock.Unlock()

	newQuote := Quote{
		Quote:  quote,
		Author: author,
		UserID: userID,
	}

	quoteIndex := -1
	if database.QuoteGraveyard == nil {
		database.QuoteGraveyard = []int{}
	}
	if len(database.QuoteGraveyard) != 0 {
		quoteIndex = database.QuoteGraveyard[0]
		database.QuoteGraveyard = database.QuoteGraveyard[1:]
	}

	if quoteIndex != -1 {
		database.Quotes[quoteIndex] = newQuote
	} else {
		database.Quotes = append(database.Quotes, newQuote)
		quoteIndex = len(database.Quotes) - 1
	}

	// Save by username as well
	_, ok = database.MapFromAuthorToQuoteIndices[strings.ToLower(author)]
	if !ok {
		database.MapFromAuthorToQuoteIndices[strings.ToLower(author)] = []int{}
	}
	database.MapFromAuthorToQuoteIndices[strings.ToLower(author)] = append(database.MapFromAuthorToQuoteIndices[strings.ToLower(author)], quoteIndex)

	return quoteIndex
}

func RemoveQuote(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if !strings.Contains(m.Content, "!removequote") {
		return
	}

	split := strings.Split(m.Content, " ")
	if len(split) != 2 {
		// Ok they put in some true garbage
		util.SendSelfDestructingMessage(s, m.ChannelID, "You must specify a quote id (its a number) like !quote 5", 5*time.Second)
		return
	}
	quoteInt, err := strconv.Atoi(split[1])
	if err != nil {
		// They put in more garbage
		util.SendSelfDestructingMessage(s, m.ChannelID, "You must specify a quote id (its a number) like !quote 5", 5*time.Second)
		return
	}

	database, ok := GuildIDToQuoteDatabase[m.GuildID]
	if !ok {
		newDatabase := &QuoteDatabase{
			Quotes:                      []Quote{},
			MapFromAuthorToQuoteIndices: map[string][]int{},
			QuoteGraveyard:              []int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[m.GuildID] = newDatabase
		database = newDatabase
	}
	database.Lock.Lock()
	defer database.Lock.Unlock()

	OriginalQuote := database.Quotes[quoteInt]
	// Remove from that authors history
	AuthorIndices, ok := database.MapFromAuthorToQuoteIndices[strings.ToLower(OriginalQuote.Author)]
	if ok {
		// Ok well its technically sorted but I wont rely on that, just hit the entire array
		new := []int{}
		for _, index := range AuthorIndices {
			if index != quoteInt {
				new = append(new, index)
			}
		}
		database.MapFromAuthorToQuoteIndices[strings.ToLower(OriginalQuote.Author)] = new
	}

	database.Quotes[quoteInt] = Quote{
		Quote: DeletedQuoteString,
	}

	if database.QuoteGraveyard == nil {
		database.QuoteGraveyard = []int{}
	}

	database.QuoteGraveyard = append(database.QuoteGraveyard, quoteInt)
	util.SendSelfDestructingMessage(s, m.ChannelID, fmt.Sprintf("Quote %d deleted successfully", quoteInt), 5*time.Second)
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
			QuoteGraveyard:              []int{},
			Lock:                        &sync.Mutex{},
		}
		GuildIDToQuoteDatabase[guildID] = newDatabase
		database = newDatabase
	}

	database.Lock.Lock()
	defer database.Lock.Unlock()

	totalQuotes := len(database.Quotes)
	if totalQuotes == 0 {
		err := util.SendSelfDestructingMessage(s, m.ChannelID, "This server has no saved quotes yet!", 10*time.Second)
		if err != nil {
			log.Printf("error sending message for random quote %v", err)
		}
		return
	}

	// Random quote
	if m.Content == "!quote" {
		database.SendQuote(s, m.ChannelID, rand.Intn(totalQuotes), totalQuotes)
		return
	}

	split := strings.SplitN(m.Content, " ", 2)
	if len(split) != 2 {
		// Ok they put in some true garbage
		util.SendSelfDestructingMessage(s, m.ChannelID, "You must specify a quote id (its a number) or a name like !quote 5 or !quote jesus", 5*time.Second)
		return
	}
	quoteInt, err := strconv.Atoi(split[1])
	if err == nil {
		database.SendQuote(s, m.ChannelID, quoteInt, totalQuotes)
		return
	}
	// Attempt to find the user?
	authorQuoteIndices, ok := database.MapFromAuthorToQuoteIndices[strings.ToLower(split[1])]
	if ok {
		database.SendQuote(s, m.ChannelID, authorQuoteIndices[rand.Intn(len(authorQuoteIndices))], totalQuotes)
		return
	}
	// Maybe its a mention?
	userID := strings.TrimSuffix(strings.TrimPrefix(split[1], "<@"), ">")
	user, err := s.User(userID)
	if err == nil {
		authorQuoteIndices, ok := database.MapFromAuthorToQuoteIndices[strings.ToLower(user.Username)]
		if ok {
			database.SendQuote(s, m.ChannelID, authorQuoteIndices[rand.Intn(len(authorQuoteIndices))], totalQuotes)
			return
		}
	}
	// Nothing we can do
	util.SendSelfDestructingMessage(s, m.ChannelID, "You must specify a quote id (its a number) or a name like !quote 5 or !quote jesus", 5*time.Second)

}

func (d *QuoteDatabase) SendQuote(s *disc.Session, ChannelID string, index int, totalQuotes int) {

	if index >= totalQuotes {
		util.SendSelfDestructingMessage(s, ChannelID, fmt.Sprintf("Sorry we only have up to quote %d", totalQuotes-1), 5*time.Second)
		return
	}

	// Check if we can mention this author
	quote := d.Quotes[index]
	author := quote.Author
	user, err := s.User(quote.UserID)
	if err == nil && user != nil {
		author = user.Mention()
	}

	_, err = s.ChannelMessageSend(ChannelID, fmt.Sprintf("[#%d]: ```%s``` -%s", index, quote.Quote, author))
	if err != nil {
		log.Printf("error sending message for random quote %v", err)
	}
}

func (d *QuoteDatabase) SendRandomQuote(s *disc.Session, ChannelID string, totalQuotes int) {
	for i := 0; i < 10; i++ {
		index := rand.Intn(totalQuotes)

		// Check if we can mention this author
		quote := d.Quotes[index]
		author := quote.Author
		if quote.Quote == DeletedQuoteString {
			continue // Dont random a deleted quote
		}
		user, err := s.User(quote.UserID)
		if err == nil && user != nil {
			author = user.Mention()
		}

		_, err = s.ChannelMessageSend(ChannelID, fmt.Sprintf("[#%d]: ```%s``` -%s", index, quote.Quote, author))
		if err != nil {
			log.Printf("error sending message for random quote %v", err)
		}
		return
	}
}
