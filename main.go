package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := InitDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	log.Printf("Bot started: @%s", bot.Self.UserName)

	go StartReminder(bot, db)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			HandleCallback(bot, db, update.CallbackQuery)
			continue
		}
		if update.Message == nil {
			continue
		}
		msg := update.Message
		userID := msg.From.ID

		if msg.IsCommand() {
			switch msg.Command() {
			case "start":
				HandleStart(bot, msg, userID)
			case "add":
				HandleAdd(bot, msg, userID)
			case "list":
				HandleList(bot, db, msg, userID)
			case "update":
				HandleUpdate(bot, db, msg, userID)
			case "delete":
				HandleDelete(bot, db, msg, userID)
			}
		} else {
			HandleTextMessage(bot, db, msg, userID)
		}
	}
}
