package tui

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	BuzzIn          key.Binding
	StartReading    key.Binding
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.FinishReading, k.CorrectAnswer, k.IncorrectAnswer, k.BuzzIn, k.StartReading}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type model struct {
	cfg         *config.Config
	webhookCh   <-chan webhook.Data
	stopWaiting context.CancelFunc

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

func (m model) transitionStartReading() (tea.Model, tea.Cmd) {
	log.Print("transitionStartReading")
	m.reading = true
	m.keymap.FinishReading.SetEnabled(true)
	m.keymap.CorrectAnswer.SetEnabled(false)
	m.keymap.IncorrectAnswer.SetEnabled(false)
	m.keymap.BuzzIn.SetEnabled(false)
	m.keymap.StartReading.SetEnabled(false)
	return m, nil
}

// Create a Cmd that will send a Msg of type webhook.Data after the webserver
// receives a buzzer webhook request, and a new Model that is ready to recieve
// the webhook.Data message
func (m model) transitionWaitForBuzzer() (tea.Model, tea.Cmd) {
	log.Print("transitionWaitForBuzzer")
	ctx, stopWaiting := context.WithCancel(context.Background())
	m.stopWaiting = stopWaiting
	m.reading = false
	m.playerAnswering = nil
	m.keymap.FinishReading.SetEnabled(false)
	m.keymap.CorrectAnswer.SetEnabled(false)
	m.keymap.IncorrectAnswer.SetEnabled(false)
	m.keymap.BuzzIn.SetEnabled(true)
	m.keymap.StartReading.SetEnabled(true)
	return m, func() tea.Msg {
		log.Println("(waiting Cmd) waiting for buzzer...")
		select {
		case data := <-m.webhookCh:
			log.Printf("(waiting Cmd) got webhook data: %+v", data)
			return data
		case <-ctx.Done():
			log.Printf("(waiting Cmd) stopped waiting: %v", ctx.Err())
			return nil
		}
	}
}

func (m model) transitionBuzzIn(player *config.Player) (tea.Model, tea.Cmd) {
	log.Print("transitionBuzzIn")
	m.playerAnswering = player
	m.answerTimer = timer.New(
		time.Duration(m.cfg.AnswerTimeoutSeconds)*time.Second,
		timer.WithInterval(100*time.Millisecond),
	)
	m.buzzedIn[player] = struct{}{}
	m.keymap.CorrectAnswer.SetEnabled(true)
	m.keymap.IncorrectAnswer.SetEnabled(true)
	m.keymap.BuzzIn.SetEnabled(false)
	return m, m.answerTimer.Init()
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
			return m.transitionWaitForBuzzer()

		case key.Matches(msg, m.keymap.CorrectAnswer):
			if m.playerAnswering == nil {
				log.Fatal("error: 'CorrectAnswer' keymap matched while player isn't buzzed-in")
			}

			// Answer was correct, go back to reading a clue
			log.Printf("'%s' answered correctly", m.playerAnswering.Name)
			return m.transitionStartReading()

		case key.Matches(msg, m.keymap.IncorrectAnswer):
			if m.playerAnswering == nil {
				log.Fatal("error: 'IncorrectAnswer' keymap matched while player isn't buzzed-in")
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
				return m.transitionStartReading()
			} else {
				log.Println("Some players have not buzzed in yet")
				return m.transitionWaitForBuzzer()
			}

		case key.Matches(msg, m.keymap.BuzzIn):
			if m.playerAnswering != nil {
				log.Fatal("error: buzz-in while there is already a player answering")
			}
			buttonId, err := strconv.Atoi(msg.String())
			if err != nil {
				log.Fatalf("error: unable to convert buzzer number to int: %v", err)
			}
			player, ok := m.cfg.PlayerForButtonId(buttonId)
			if !ok {
				log.Printf("received manual buzz-in with invalid ButtonId: %d", buttonId)
				return m, nil
			}
			if _, buzzedIn := m.buzzedIn[player]; buzzedIn {
				log.Printf("player has already buzzed in: %+v", player)
				return m.transitionWaitForBuzzer()
			}
			log.Printf("Buzzed in manually: %+v", player)
			m.stopWaiting()
			m.stopWaiting = nil
			return m.transitionBuzzIn(player)
		case key.Matches(msg, m.keymap.StartReading):
			if m.stopWaiting != nil {
				m.stopWaiting()
				m.stopWaiting = nil
			}
			log.Printf("Going back to reading")
			return m.transitionStartReading()
		}

	case webhook.Data:
		if m.playerAnswering != nil {
			log.Fatal("error: buzz-in while there is already a player answering")
		}
		player, ok := m.cfg.PlayerForButtonId(msg.ButtonId)
		if !ok {
			log.Printf("error: received webhook.Data message with invalid ButtonId: %d", msg.ButtonId)
			return m, tea.Quit
		}
		if _, buzzedIn := m.buzzedIn[player]; buzzedIn {
			log.Printf("player has already buzzed in: %+v", player)
			return m.transitionWaitForBuzzer()
		}
		log.Printf("Player buzzed in: %+v", player)
		return m.transitionBuzzIn(player)
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
