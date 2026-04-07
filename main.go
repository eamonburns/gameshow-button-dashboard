package main

import (
	"log"
	"os"

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
	go webhook.StartListening(addr, webhookId, webhookCh)
	log.Printf("Started listening for webhooks on %s\n", addr)

	err = tui.Start(webhookCh)
	if err != nil {
		log.Fatalf("An error occured while running the TUI: %v\n", err)
	}
}
