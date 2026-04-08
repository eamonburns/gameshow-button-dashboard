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

type keymap struct {
	Quit            key.Binding
	FinishReading   key.Binding
	CorrectAnswer   key.Binding
	IncorrectAnswer key.Binding
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.FinishReading, k.CorrectAnswer, k.IncorrectAnswer}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type model struct {
	cfg       *config.Config
	webhookCh <-chan webhook.Data

	reading bool

	// The player that is currently answering the question
	// `nil` if waiting for a player to buzz-in
	playerAnswering *config.Player
	answerTimer     timer.Model
	// Players that have buzzed in are added to this "set"
	buzzedIn map[*config.Player]struct{}

	keymap keymap
	help   help.Model
}

// Create a Cmd that will send a Msg of type webhook.Data after the webserver
// receives a buzzer webhook request, and a new Model that is ready to recieve
// the webhook.Data message
func (m model) waitForBuzzer() (tea.Model, tea.Cmd) {
	m.reading = false
	m.playerAnswering = nil
	m.keymap.FinishReading.SetEnabled(false)
	m.keymap.CorrectAnswer.SetEnabled(false)
	m.keymap.IncorrectAnswer.SetEnabled(false)
	return m, func() tea.Msg {
		log.Println("waiting for buzzer...")
		data := <-m.webhookCh
		log.Println("got webhook data")
		return data
	}
}

func (m model) startReading() (tea.Model, tea.Cmd) {
	m.reading = true
	m.keymap.FinishReading.SetEnabled(true)
	m.keymap.CorrectAnswer.SetEnabled(false)
	m.keymap.IncorrectAnswer.SetEnabled(false)
	return m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		case key.Matches(msg, m.keymap.FinishReading):
			m.buzzedIn = make(map[*config.Player]struct{}, len(m.cfg.Players))
			return m.waitForBuzzer()

		case key.Matches(msg, m.keymap.CorrectAnswer):
			if m.playerAnswering == nil {
				log.Printf("error: this 'CorrectAnswer' keymap matched while player isn't buzzed-in")
				break
			}

			// Answer was correct, go back to reading a clue
			log.Printf("'%s' answered correctly", m.playerAnswering.Name)
			return m.startReading()

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
				return m.startReading()
			} else {
				// FIXME: There is a possibility that the last player could
				// just answer the question without buzzing in, because they
				// don't have anyone to race for the answer. There isn't
				// currently a way to override the behavior of having to
				// buzz-in before being able to move on, so in that case they
				// would have to buzz-in after already answering the question
				// and the host would have to accept/reject their answer
				log.Println("Some players have not buzzed in yet")
				return m.waitForBuzzer()
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
			return m.waitForBuzzer()
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

func (m model) View() tea.View {
	var s strings.Builder

	if m.reading {
		// Reading clue
		fmt.Fprint(&s, "Reading clue...\n")
	} else {
		// Buzzing-in and answering
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
	}

	fmt.Fprintf(&s, "\n%s\n", m.help.View(m.keymap))

	v := tea.NewView(s.String())
	v.AltScreen = true
	return v
}
