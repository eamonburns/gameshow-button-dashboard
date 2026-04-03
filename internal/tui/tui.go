package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

func Start(webhookCh <-chan webhook.Data) error {
	p := tea.NewProgram(initialModel(webhookCh))
	_, err := p.Run()
	return err
}
