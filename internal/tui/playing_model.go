package tui

import (
	"fmt"
	"log"

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

			// Answer was incorrect, go back to listening for buzzer
			log.Printf("'%s' answered incorrectly", m.playerAnswering.Name)
			m.playerAnswering = nil
			// TODO: Go back to reading if all players have buzzed in
			return m, m.waitForBuzzer()
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
	if m.playerAnswering == nil {
		return tea.NewView("Waiting for buzzer...")
	}

	return tea.NewView(fmt.Sprintf("'%s' is answering", m.playerAnswering.Name))
}
