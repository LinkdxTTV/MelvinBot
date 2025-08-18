package util

import (
	"fmt"
	"log"
	"time"

	disc "github.com/bwmarrin/discordgo"
)

// Since this has a sleep, we should run it in a goroutine
func SendSelfDestructingMessage(s *disc.Session, channelID string, content string, duration time.Duration) {
	go func() {
		content += fmt.Sprintf(" [This message will self delete in %s]", duration)
		msg, err := s.ChannelMessageSend(channelID, content)
		if err != nil {
			log.Printf("failed to send message: %v", err)
		}
		time.Sleep(duration)
		err = s.ChannelMessageDelete(channelID, msg.ID)
		if err != nil {
			log.Printf("failed to delete message: %v", err)
		}
	}()
}
