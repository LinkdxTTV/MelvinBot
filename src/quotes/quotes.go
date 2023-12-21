package quotes

import (
	"fmt"
	"log"
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
}

func (q *Quote) String() string {
	return fmt.Sprintf("%s -%s", q.Quote, q.Author)
}

var GuildIDToQuoteDatabase = map[string]QuoteDatabase{}

func LookForQuoteReact(s *disc.Session, m *disc.MessageReactionAdd) {
	if m.MessageReaction.Emoji.Name != "💬" {
		return
	}

	err := s.ChannelMessagePin(m.ChannelID, m.MessageID)
	if err != nil {
		log.Printf("error pinning: %v", err)
	}
}

func addQuote(guildID string) {}
