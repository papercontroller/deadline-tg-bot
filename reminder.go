package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type reminderSpec struct {
	flag     string
	duration time.Duration
	label    string
}

var reminderSchedule = []reminderSpec{
	{"24h", 24 * time.Hour, "24 hours"},
	{"3h", 3 * time.Hour, "3 hours"},
}

func StartReminder(bot *tgbotapi.BotAPI, db *DB) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	checkReminders(bot, db)
	for range ticker.C {
		checkReminders(bot, db)
	}
}

func checkReminders(bot *tgbotapi.BotAPI, db *DB) {
	for _, spec := range reminderSchedule {
		deadlines, err := db.GetPendingReminders(spec.duration, spec.flag)
		if err != nil {
			log.Printf("reminder query error (%s): %v", spec.flag, err)
			continue
		}
		for _, d := range deadlines {
			text := fmt.Sprintf(
				"⏰ *Reminder:* %s\n📅 Due: %s - in *%s*",
				d.Text,
				d.DeadlineAt.Format("02.01.2006 15:04"),
				spec.label,
			)
			if err := sendWithRetry(bot, d.UserID, text); err != nil {
				log.Printf("reminder for deadline %d failed after retries: %v", d.ID, err)
				continue
			}
			if err := db.MarkReminded(d.ID, spec.flag); err != nil {
				log.Printf("failed to mark reminded %d/%s: %v", d.ID, spec.flag, err)
			}
		}
	}
}

func sendWithRetry(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	delays := []time.Duration{0, 5 * time.Second, 10 * time.Second}
	for attempt, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		if _, err := bot.Send(msg); err == nil {
			return nil
		} else if attempt < len(delays)-1 {
			log.Printf("send attempt %d failed, retrying: %v", attempt+1, err)
		} else {
			return err
		}
	}
	return nil
}
