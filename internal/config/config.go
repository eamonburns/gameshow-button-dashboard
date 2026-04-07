package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

// Static configuration, loaded at the start of the program
type Config struct {
	Players []*Player
}

type Player struct {
	Name     string `json:"name"`
	ButtonId int    `json:"button_id"`
}

func Load(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	if len(cfg.Players) == 0 {
		return nil, errors.New("players is empty")
	}
	seenButtonIds := make(map[int]int)
	seenNames := make(map[string]int)
	for i, player := range cfg.Players {
		if player == nil {
			return nil, fmt.Errorf("players[%d] is null", i)
		}
		if player.ButtonId == 0 {
			return nil, fmt.Errorf("players[%d].button_id is 0", i)
		}
		if player.Name == "" {
			return nil, fmt.Errorf("players[%d].name is empty", i)
		}
		if seenIdx, seen := seenButtonIds[player.ButtonId]; seen {
			return nil, fmt.Errorf("players[%d].button_id is the same as players[%d].button_id", i, seenIdx)
		} else {
			seenButtonIds[player.ButtonId] = i
		}
		if seenIdx, seen := seenNames[player.Name]; seen {
			return nil, fmt.Errorf("players[%d].name is the same as players[%d].name", i, seenIdx)
		} else {
			seenNames[player.Name] = i
		}
	}

	return &cfg, nil
}

func (sets Config) Format(f fmt.State, verb rune) {
	fmt.Fprint(f, "Config{ Players: [ ")
	for _, player := range sets.Players {
		fmt.Fprintf(f, "%+v ", player)
	}
	fmt.Fprint(f, "] }")
}
