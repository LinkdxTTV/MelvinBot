package nisha

import (
	"MelvinBot/src/util"
	"fmt"
	"strings"
	"time"

	disc "github.com/bwmarrin/discordgo"
)

func DidSomebodySaySex(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "sex") {
		s.ChannelMessageSend(m.ChannelID, "did somebody say sex???")
	}
}

func ThisIsNotADvd(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if m.Content != "!stop" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "STOP! STOP! STOP! This is NOT a DVD. This is NOT A DVD. THIS IS NOT A DVD. This is a BACKER CARD. It's a CARD for COLLECTORS. This is a MOVIE CARD. THIS IS NOT A DVD. STOP! READ. READ THE DESCRIPTION.")

}

func GeorgeCarlin(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if m.Content != "!rsbs" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "RATSHIT BATSHIT DIRTY OLD TWAT 69 ASSHOLES TIED IN A KNOT HOORAY LIZARD SHIT FUCK")
}

func Tetazoo(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "tetazoo") {
		s.ChannelMessageSend(m.ChannelID, "TETAZOO IS NOT A HIVEMIND")
	}
}

func Glounge(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "where are you") {
		s.ChannelMessageSend(m.ChannelID, "update tetazoo glounge")
	}
}

func Iiwii(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.Content != "!iiwii" {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "it EEEEEEES what it eees")
}

func Lethimcook(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "cook") {
		s.ChannelMessageSend(m.ChannelID, "https://i.kym-cdn.com/entries/icons/original/000/041/943/1aa1blank.png")
	}
}

func Miami(s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	if strings.Contains(strings.ToLower(m.Message.Content), "miami") || strings.Contains(strings.ToLower(m.Message.Content), "spring break") {
		year := []rune(fmt.Sprint(time.Now().Year()))
		year[1] = 'k'
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("SPRING BREAK MIAMI %v WOOOOOOOO", string(year)))
	}
}

func KillDamian (s *disc.Session, m *disc.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return // it me
	}

	if m.GuildID != util.Wolfcord_GuildID {
		return // only for nisha's discord
	}

	// if damian replies to erik specifically
	if m.Message.Reference != nil {
		referencedMsg, err := s.ChannelMessage(m.ChannelID, m.Message.Reference.MessageID)
		if err == nil && referencedMsg.Author.ID == "ejporter" && m.Author.ID == "damianlx" {
			postPicture(s, m.ChannelID)
		}
	}

	// if damian replies after erik
	messages, err := s.ChannelMessages(m.ChannelID, 2, m.ID, "", "")
	if err == nil && len(messages) > 1 {
		lastMessage := messages[1] 
		if lastMessage.Author.ID == "ejporter" && m.Author.ID == "damianlx" {
			postPicture(s, m.ChannelID)
		}
	}
}

func postPicture(s *disc.Session, channelID string) {
	file, err := os.Open("./assets/lookattheflowers.jpg") 
	if err != nil {
		s.ChannelMessageSend(channelID, "Error: Unable to load image.")
		return
	}
	defer file.Close()

	// Send the image as an attachment
	s.ChannelFileSend(channelID, "lookattheflowers.jpg", file)
}