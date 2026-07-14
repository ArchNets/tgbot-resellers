package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/bot"
	"reseller-bot/pkg/config"
	"reseller-bot/pkg/db"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	dbPath := flag.String("db", "bot.db", "Path to database file")
	flag.Parse()

	log.Println("Starting Standalone Reseller Telegram Bot...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Database
	database, err := db.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		log.Println("Closing database...")
		database.Close()
	}()

	// Initialize Backend client
	backendClient := backend.NewClient(cfg.BackendURL, cfg.ResellerAPIKey)

	// Initialize Bot
	telegramBot, err := bot.NewBot(cfg, database, backendClient)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram Bot: %v", err)
	}

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start bot updates polling
	go telegramBot.Start(ctx)

	// Block until context is cancelled (signal received)
	<-ctx.Done()
	log.Println("Shutdown signal received. Exiting gracefully...")
}
