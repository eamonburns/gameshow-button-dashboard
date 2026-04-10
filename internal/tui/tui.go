package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

func initialModel(cfg *config.Config, webhookCh <-chan webhook.Data) model {
	return model{
		cfg:       cfg,
		webhookCh: webhookCh,
		reading:   true,

		keymap: keymap{
			Quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			FinishReading: key.NewBinding(
				key.WithKeys("space"),
				key.WithHelp("space", "finish reading"),
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
			BuzzIn: key.NewBinding(
				// NOTE: I might want to change this to use the actual button IDs of the players
				key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
				key.WithHelp("1-9", "buzz-in"),
				key.WithDisabled(), // Will be enabled when waiting for buzzer
			),
			StartReading: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back to reading"),
				key.WithDisabled(), // Will be enabled when not reading
			),
		},
		help: help.New(),
	}
}

func Start(cfg *config.Config, webhookCh <-chan webhook.Data) error {
	p := tea.NewProgram(initialModel(cfg, webhookCh))
	_, err := p.Run()
	return err
}
