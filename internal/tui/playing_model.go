package tui

import (
	"fmt"
	"log"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

type playingModel struct {
	cfg       *config.Config
	webhookCh <-chan webhook.Data

	// The player that is currently answering the question
	// `nil` if waiting for a player to buzz-in
	playerAnswering *config.Player
	// Players that have buzzed in are added to this "set"
	buzzedIn map[*config.Player]struct{}
}

// Create a new playingModel and also a tea.Cmd to wait for buzzer presses
func newPlayingModel(cfg *config.Config, webhookCh <-chan webhook.Data) (playingModel, tea.Cmd) {
	m := playingModel{
		cfg:       cfg,
		webhookCh: webhookCh,
		buzzedIn:  make(map[*config.Player]struct{}),
	}
	return m, m.waitForBuzzer()
}

// Create a Cmd that will send a Msg of type webhook.Data after the webserver
// receives a buzzer webhook request
func (m playingModel) waitForBuzzer() tea.Cmd {
	return func() tea.Msg {
		log.Printf("waiting for buzzer...")
		data := <-m.webhookCh
		log.Printf("got webhook data")
		return data
	}
}

func (m playingModel) Init() tea.Cmd {
	return nil
}

func (m playingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.playerAnswering == nil {
				break
			}

			// Answer was correct, go back to reading a clue
			log.Printf("'%s' answered correctly", m.playerAnswering.Name)
			return readingModel{
				cfg:       m.cfg,
				webhookCh: m.webhookCh,
			}, nil
		case "backspace":
			if m.playerAnswering == nil {
				break
			}

			log.Printf("'%s' answered incorrectly", m.playerAnswering.Name)
			allBuzzedIn := true
			for _, player := range m.cfg.Players {
				if _, buzzedIn := m.buzzedIn[player]; !buzzedIn {
					allBuzzedIn = false
					break
				}
			}
			if allBuzzedIn {
				log.Println("All players answered incorrectly")
				return readingModel{
					cfg:       m.cfg,
					webhookCh: m.webhookCh,
				}, nil
			} else {
				log.Println("Some players have not buzzed in yet")
				m.playerAnswering = nil
				return m, m.waitForBuzzer()
			}
		}
	case webhook.Data:
		player, ok := m.cfg.PlayerForButtonId(msg.ButtonId)
		if !ok {
			log.Printf("error: received webhook.Data message with invalid ButtonId: %d", msg.ButtonId)
			return m, tea.Quit
		}
		if _, buzzedIn := m.buzzedIn[player]; buzzedIn {
			log.Printf("player has already buzzed in: %+v", player)
			return m, m.waitForBuzzer()
		}
		log.Printf("Player buzzed in: %+v", player)
		m.playerAnswering = player
		m.buzzedIn[player] = struct{}{}
		return m, nil
	}

	return m, nil
}

func (m playingModel) View() tea.View {
	var s strings.Builder
	fmt.Fprint(&s, "Players:\n")
	for i, player := range m.cfg.Players {
		fmt.Fprintf(&s, "%d. %s", i+1, player.Name)
		if player == m.playerAnswering {
			fmt.Fprint(&s, " (answering)\n")
		} else if _, buzzedIn := m.buzzedIn[player]; buzzedIn {
			fmt.Fprint(&s, " (buzzed-in)\n")
		} else {
			fmt.Fprint(&s, "\n")
		}
	}
	fmt.Fprint(&s, "\n")

	if m.playerAnswering == nil {
		fmt.Fprint(&s, "Waiting for buzzer...\n")
	} else {
		fmt.Fprintf(&s, "'%s' is answering...\n", m.playerAnswering.Name)
	}

	return tea.NewView(s.String())
}
