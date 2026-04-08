package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

func Start(cfg *config.Config, webhookCh <-chan webhook.Data) error {
	p := tea.NewProgram(readingModel{
		cfg:       cfg,
		webhookCh: webhookCh,
	})
	_, err := p.Run()
	return err
}
