package util

import (
	"fmt"
	"time"

	disc "github.com/bwmarrin/discordgo"
)

func SendSelfDestructingMessage(s *disc.Session, channelID string, content string, duration time.Duration) error {
	content += fmt.Sprintf(" [This message will self delete in %s]", duration)
	msg, err := s.ChannelMessageSend(channelID, content)
	if err != nil {
		return err
	}
	time.Sleep(duration)
	err = s.ChannelMessageDelete(channelID, msg.ID)
	if err != nil {
		return err
	}
	return nil
}
