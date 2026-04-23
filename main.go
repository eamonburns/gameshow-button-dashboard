package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	config "github.com/eamonburns/gameshow-button-dashboard/internal/config"
	"github.com/eamonburns/gameshow-button-dashboard/internal/tui"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

func main() {
	logFile, err := tea.LogToFile("buttons.log", "debug")
	if err != nil {
		log.Fatalf("error: unable to open log file: %v\n", err)
	}
	defer logFile.Close()
	log.Println("Starting thing")

	configPath := "config.json"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("error: unable to load config from '%s': %v\n", configPath, err)
	}
	log.Printf("Config: %+v\n", cfg)

	webhookId := os.Getenv("WEBHOOK_ID")
	if webhookId == "" {
		log.Fatalln("error: environment variable WEBHOOK_ID is not defined")
	}
	webhookCh := make(chan webhook.Data)
	addr := ":8080"
	go webhook.StartListening(addr, webhookId, cfg, webhookCh)
	log.Printf("Started listening for webhooks on %s\n", addr)

	if len(os.Args) > 1 && slices.Contains(os.Args[1:], "--log-webhooks") {
		logWebhooks(cfg, webhookCh)
		os.Exit(0)
	}

	err = tui.Start(cfg, webhookCh)
	if err != nil {
		log.Fatalf("An error occured while running the TUI: %v\n", err)
	}
}

// Log received webhooks in a loop
func logWebhooks(cfg *config.Config, webhookCh <-chan webhook.Data) {
	fmt.Println("Logging webhooks...")

	for {
		data := <-webhookCh

		now := time.Now().Format(time.DateTime)
		if player, ok := cfg.PlayerForButtonId(data.ButtonId); ok {
			fmt.Printf("%s Received webhook, player: %+v\n", now, player)
		} else {
			fmt.Printf("%s Received webhook, unknown button ID: %d\n", now, data.ButtonId)
		}
	}
}
