package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/timer"
	tea "charm.land/bubbletea/v2"

	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

type playingKeymap struct {
	Quit            key.Binding
	CorrectAnswer   key.Binding
	IncorrectAnswer key.Binding
}

func (k playingKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.CorrectAnswer, k.IncorrectAnswer}
}

func (k playingKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type playingModel struct {
	cfg       *config.Config
	webhookCh <-chan webhook.Data

	// The player that is currently answering the question
	// `nil` if waiting for a player to buzz-in
	playerAnswering *config.Player
	answerTimer     timer.Model
	// Players that have buzzed in are added to this "set"
	buzzedIn map[*config.Player]struct{}

	keymap playingKeymap
	help   help.Model
}

// Create a new playingModel and also a tea.Cmd to wait for buzzer presses
func newPlayingModel(cfg *config.Config, webhookCh <-chan webhook.Data) (playingModel, tea.Cmd) {
	m := playingModel{
		cfg:       cfg,
		webhookCh: webhookCh,
		buzzedIn:  make(map[*config.Player]struct{}),
		keymap: playingKeymap{
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			CorrectAnswer: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "accept answer"),
				key.WithDisabled(), // Will be enabled when a player has buzzed in
			),
			IncorrectAnswer: key.NewBinding(
				key.WithKeys("backspace"),
				key.WithHelp("backspace", "reject answer"),
				key.WithDisabled(), // Will be enabled when a player has buzzed in
			),
		},
		help: help.New(),
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
	case timer.TickMsg:
		var cmd tea.Cmd
		m.answerTimer, cmd = m.answerTimer.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.CorrectAnswer):
			if m.playerAnswering == nil {
				log.Printf("error: this 'CorrectAnswer' keymap matched while player isn't buzzed-in")
				break
			}

			// Answer was correct, go back to reading a clue
			log.Printf("'%s' answered correctly", m.playerAnswering.Name)
			return readingModel{
				cfg:       m.cfg,
				webhookCh: m.webhookCh,
			}, nil

		case key.Matches(msg, m.keymap.IncorrectAnswer):
			if m.playerAnswering == nil {
				log.Printf("error: this 'IncorrectAnswer' keymap matched while player isn't buzzed-in")
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
				m.keymap.CorrectAnswer.SetEnabled(false)
				m.keymap.IncorrectAnswer.SetEnabled(false)
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
		m.answerTimer = timer.New(
			time.Duration(m.cfg.AnswerTimeoutSeconds)*time.Second,
			timer.WithInterval(100*time.Millisecond),
		)
		m.buzzedIn[player] = struct{}{}
		m.keymap.CorrectAnswer.SetEnabled(true)
		m.keymap.IncorrectAnswer.SetEnabled(true)
		return m, m.answerTimer.Init()
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
	} else if m.answerTimer.Timedout() {
		fmt.Fprintf(&s, "'%s' is answering (timed out)\n", m.playerAnswering.Name)
	} else {
		fmt.Fprintf(&s, "'%s' is answering (%s)...\n", m.playerAnswering.Name, m.answerTimer.View())
	}

	fmt.Fprintf(&s, "\n%s\n", m.help.View(m.keymap))

	return tea.NewView(s.String())
}
