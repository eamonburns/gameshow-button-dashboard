package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

func initialModel(cfg *config.Config, webhookCh <-chan webhook.Data) readingModel {
	return readingModel{
		cfg:       cfg,
		webhookCh: webhookCh,
	}
}

func Start(cfg *config.Config, webhookCh <-chan webhook.Data) error {
	p := tea.NewProgram(initialModel(cfg, webhookCh))
	_, err := p.Run()
	return err
}
