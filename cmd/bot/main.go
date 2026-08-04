package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/bot"
	"reseller-bot/pkg/config"
	"reseller-bot/pkg/db"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	dbPathFlag := flag.String("db", "bot.db", "Path to database file")
	flag.Parse()

	dbPathSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "db" {
			dbPathSet = true
		}
	})

	resolvedDBPath := *dbPathFlag
	if envDBPath := os.Getenv("DB_PATH"); envDBPath != "" && !dbPathSet {
		resolvedDBPath = envDBPath
	}

	log.Println("Starting Standalone Reseller Telegram Bot...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.LogSummary()

	// Ensure parent directory for database exists (0700)
	if dir := filepath.Dir(resolvedDBPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			log.Fatalf("Failed to create database directory %s: %v", dir, err)
		}
	}

	log.Printf("Using database path: %s", resolvedDBPath)

	// Initialize Database
	database, err := db.NewDB(resolvedDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		log.Println("Closing database...")
		database.Close()
	}()

	// Initialize Backend client
	backendClient := backend.NewClient(cfg.BackendURL, cfg.ResellerAPIKey, cfg.HostMappings, cfg.InsecureSkipVerify, cfg.BotID)

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
