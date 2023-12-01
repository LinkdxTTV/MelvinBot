package store

import (
	stats "MelvinBot/src/stats"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type Storage interface {
	Put() error
	Get() error
	SyncOnTimer(time.Duration) error
}

type localStorage struct {
	filename string
}

func NewLocalStorage(filename ...string) (*localStorage, error) {
	if len(filename) > 1 {
		return nil, errors.New("cannot specify more than one filepath for local stoage")
	}

	if len(filename) == 0 {
		filename = []string{"./stats"}
	}

	return &localStorage{
		filename: filename[0],
	}, nil
}

func (s *localStorage) Put() error {
	statsAsJson, err := json.Marshal(stats.StatsPerGuild)
	if err != nil {
		return err
	}

	statsFile, err := os.Create(s.filename)
	if err != nil {
		return errors.New("could not create stats file")
	}
	_, err = statsFile.Write(statsAsJson)
	if err != nil {
		return errors.New("could not write stats to file")
	}
	return statsFile.Close()
}

func (s *localStorage) Get() error {
	bytes, err := os.ReadFile(s.filename)
	if err != nil {
		return errors.New("could not read local stats file")
	}
	newStats := stats.StatsPerGuild
	err = json.Unmarshal(bytes, &newStats)
	if err != nil {
		return errors.New(fmt.Sprintf("error unmarshaling, %v", err))
	}
	return nil
}

func (s *localStorage) SyncOnTimer(timer time.Duration) error {
	newTimer := time.NewTicker(timer)

	go func() {
		for {
			<-newTimer.C
			err := s.Put()
			if err != nil {
				log.Print("error putting stats")
			}
		}
	}()
	return nil
}
