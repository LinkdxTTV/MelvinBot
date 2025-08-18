package quotes

import (
	"MelvinBot/src/util"
	"bytes"
	"fmt"
	"io"
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
	Quote          string
	AttachmentURLs []string
	Author         string
	UserID         string
}

func (q *Quote) String() string {
	return fmt.Sprintf("%s -@%s", q.Quote, q.Author)
}

var GuildIDToQuoteDatabase = map[string]*QuoteDatabase{}

func AddQuote(s *disc.Session, m *disc.MessageReactionAdd) {
	if m.MessageReaction.Emoji.Name != "💬" {
		return
	}

	message, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		// We cant find the message that was just reacted to?
		return
	}
	if message.Author.ID == s.State.User.ID {
		// Disallows melvinbot from saving quotes from himself
		return
	}

	// Dedupe quotes
	for _, reaction := range message.Reactions {
		if reaction.Emoji.Name == "💬" {
			if reaction.Count > 1 {
				return
			}
		}
	}

	// Disallow abuse via reacting and unreacting quote over and over.. but this only checks the last quote
	guildID := m.GuildID
	db, ok := GuildIDToQuoteDatabase[guildID]
	if ok {
		lastQuote := db.Quotes[len(db.Quotes)-1]
		if lastQuote.Quote == message.Content {
			return
		}
	}

	// Check for attachments
	attachments := []string{}

	for _, attachment := range message.Attachments {
		attachments = append(attachments, attachment.URL)
	}

	newQuoteID := AddQuoteToDatabase(guildID, message.Content, attachments, message.Author.Username, message.Author.ID)
	// Finally ack
	maybeContainsAttachments := ""
	if len(attachments) > 0 {
		maybeContainsAttachments += "[Contains Attachments]"
	}

	messageContent := ""
	if len(message.Content) > 0 {
		messageContent = fmt.Sprintf("```%s```", message.Content)
	}

	util.SendSelfDestructingMessage(s, m.ChannelID, fmt.Sprintf("Added quote [#%d]: %s %s -%s", newQuoteID, messageContent, maybeContainsAttachments, message.Author.Username), 10*time.Second)

}

func AddQuoteToDatabase(guildID string, quote string, attachmentURLs []string, author string, userID string) int {
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
		Quote:          quote,
		AttachmentURLs: attachmentURLs,
		Author:         author,
		UserID:         userID,
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
		util.SendSelfDestructingMessage(s, m.ChannelID, "This server has no saved quotes yet!", 10*time.Second)
		return
	}

	// Random quote
	if m.Content == "!quote" {
		database.SendRandomQuote(s, m.ChannelID, totalQuotes)
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

	// allow getting all quotes
	if strings.ToLower(split[1]) == "all" {
		database.SendAllQuotesAsAttachment(s, m.ChannelID)
		return
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

	attachmentURLS := ""
	for _, URL := range quote.AttachmentURLs {
		attachmentURLS += "\n"
		attachmentURLS += URL
	}

	var body string
	if len(quote.Quote) > 0 {
		body = fmt.Sprintf("```%s```", quote.Quote)
	}
	_, err = s.ChannelMessageSend(ChannelID, fmt.Sprintf("[#%d]: %s %s\n-%s", index, body, attachmentURLS, author))
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

func (d *QuoteDatabase) SendAllQuotesAsAttachment(s *disc.Session, channelID string) {
	var quoteBuffer bytes.Buffer

	for i, quote := range d.Quotes {
		if quote.Quote == DeletedQuoteString {
			continue
		}
		tempQuote := fmt.Sprintf("%d : %s : %s ", i, quote.Author, quote.Quote)
		if len(quote.AttachmentURLs) != 0 {
			tempQuote += fmt.Sprintf("(Attachments: %s )", strings.Join(quote.AttachmentURLs, ", "))
		}
		tempQuote += "\r\n" // Add windows style newlines
		quoteBuffer.WriteString(tempQuote)
	}

	var reader io.Reader = &quoteBuffer
	filemsg, err := s.ChannelFileSend(channelID, "quotes.txt", reader)
	if err != nil {
		log.Printf("error sending all quotes %v", err)
		return
	}

	util.SendSelfDestructingMessage(s, channelID, "Deleting this file in 30 seconds", 30*time.Second)
	go func() {
		time.Sleep(30 * time.Second)
		err = s.ChannelMessageDelete(channelID, filemsg.ID)
		if err != nil {
			log.Printf("error deleting all quotes file %v", err)
		}
	}()
}
