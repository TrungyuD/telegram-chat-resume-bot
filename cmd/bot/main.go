package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/bot"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/config"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/dashboard"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

func main() {
	// Load .env file (optional, env vars take precedence)
	_ = godotenv.Load()

	// Initialize data directories
	if err := storage.InitDataDirs(); err != nil {
		log.Fatalf("Failed to init data dirs: %v", err)
	}

	// Load config
	cfg := config.LoadGlobalConfig()
	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required (set in .env or environment)")
	}

	// Create and start bot
	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	dashboardCtx, cancelDashboard := context.WithCancel(context.Background())
	defer cancelDashboard()
	go func() {
		addr := fmt.Sprintf(":%d", cfg.WebPort)
		log.Printf("Dashboard starting on %s", addr)
		if err := dashboard.StartServer(dashboardCtx, addr, cfg); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard stopped with error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down...")
		cancelDashboard()
		b.Stop()
	}()

	log.Println("Bot starting...")
	b.Start()
}
