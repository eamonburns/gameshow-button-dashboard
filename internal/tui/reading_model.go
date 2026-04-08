package tui

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

const FINISH_READING_KEY = "space"

type readingModel struct {
	cfg       *config.Config
	webhookCh <-chan webhook.Data
}

func (m readingModel) Init() tea.Cmd {
	return nil
}

func (m readingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case FINISH_READING_KEY:
			log.Println("Finished reading clue")
			return newPlayingModel(m.cfg, m.webhookCh)
		}
	}

	return m, nil
}

func (m readingModel) View() tea.View {
	return tea.NewView("Reading clue. Press " + FINISH_READING_KEY + " to listen for button presses.")
}
