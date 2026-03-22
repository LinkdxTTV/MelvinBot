package store

import (
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
	filename   string
	input      any
	keepBackup bool
}

func NewLocalStorage(input any, backup bool, filename ...string) (*localStorage, error) {
	if len(filename) > 1 {
		return nil, errors.New("cannot specify more than one filepath for local stoage")
	}

	if len(filename) == 0 {
		filename = []string{"./stats"}
	}

	return &localStorage{
		filename:   filename[0],
		input:      input,
		keepBackup: backup,
	}, nil
}

func (s *localStorage) Put() error {
	statsAsJson, err := json.Marshal(s.input)
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
	err = statsFile.Close()
	if err != nil {
		return err
	}

	if s.keepBackup {
		statsFileBackup, err := os.Create(s.filename + "_backup")
		if err != nil {
			return errors.New("could not create stats backup file")
		}
		_, err = statsFileBackup.Write(statsAsJson)
		if err != nil {
			return errors.New("could not write stats backup to file")
		}
		err = statsFileBackup.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *localStorage) Get() error {
	bytes, err := os.ReadFile(s.filename)
	if err != nil {
		return errors.New("could not read local stats file")
	}
	newStats := s.input
	err = json.Unmarshal(bytes, &newStats)
	if err != nil {
		return fmt.Errorf("error unmarshaling, %v", err)
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
