package main

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func send(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func sendKb(bot *tgbotapi.BotAPI, chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb
	return bot.Send(msg)
}

func deleteMsg(bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	bot.Request(tgbotapi.NewDeleteMessage(chatID, msgID))
}

func HandleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, userID int64) {
	resetState(userID)
	send(bot, msg.Chat.ID, `👋 *Deadline Bot*

Track all your deadlines in one place.

*Commands:*
/add - add a deadline
/list - show upcoming deadlines
/update - edit a deadline
/delete - delete all your deadlines`)
}

func HandleAdd(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, userID int64) {
	setState(userID, UserState{Step: StepAddText})
	send(bot, msg.Chat.ID, "📝 What's the deadline about?")
}

func HandleList(bot *tgbotapi.BotAPI, db *DB, msg *tgbotapi.Message, userID int64) {
	deadlines, err := db.ListDeadlines(userID)
	if err != nil {
		send(bot, msg.Chat.ID, "❌ Could not fetch deadlines.")
		return
	}
	if len(deadlines) == 0 {
		send(bot, msg.Chat.ID, "📭 No upcoming deadlines. Use /add to add one.")
		return
	}
	var sb strings.Builder
	sb.WriteString("📋 *Your upcoming deadlines:*\n\n")
	for _, d := range deadlines {
		daysLeft := int(time.Until(d.DeadlineAt).Hours() / 24)
		var urgency string
		switch {
		case daysLeft == 0:
			urgency = " ⚠️ today"
		case daysLeft == 1:
			urgency = " ⚠️ tomorrow"
		default:
			urgency = fmt.Sprintf(" (%d days)", daysLeft)
		}
		sb.WriteString(fmt.Sprintf("*%s* — %s%s\n", d.Text, d.DeadlineAt.Format("02.01.2006"), urgency))
	}
	send(bot, msg.Chat.ID, sb.String())
}

func HandleUpdate(bot *tgbotapi.BotAPI, db *DB, msg *tgbotapi.Message, userID int64) {
	deadlines, err := db.ListDeadlines(userID)
	if err != nil {
		send(bot, msg.Chat.ID, "❌ Could not fetch deadlines.")
		return
	}
	if len(deadlines) == 0 {
		send(bot, msg.Chat.ID, "📭 No deadlines to update.")
		return
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, d := range deadlines {
		label := fmt.Sprintf("%s - %s", truncate(d.Text, 28), d.DeadlineAt.Format("02.01"))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("upd_select:%d", d.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
	))
	setState(userID, UserState{Step: StepUpdateSelect})
	sendKb(bot, msg.Chat.ID, "✏️ Which deadline to update?", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func HandleDelete(bot *tgbotapi.BotAPI, db *DB, msg *tgbotapi.Message, userID int64) {
	deadlines, err := db.ListDeadlines(userID)
	if err != nil {
		send(bot, msg.Chat.ID, "❌ Could not fetch deadlines.")
		return
	}
	if len(deadlines) == 0 {
		send(bot, msg.Chat.ID, "📭 No deadlines to delete.")
		return
	}
	selected := make(map[int64]bool)
	setState(userID, UserState{Step: StepDeleteSelect, SelectedIDs: selected})
	sendKb(bot, msg.Chat.ID, "🗑 Select deadlines to delete:", deleteKeyboard(deadlines, selected))
}

func deleteKeyboard(deadlines []Deadline, selected map[int64]bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, d := range deadlines {
		prefix := "☐ "
		if selected[d.ID] {
			prefix = "✅ "
		}
		label := fmt.Sprintf("%s%s — %s", prefix, truncate(d.Text, 24), d.DeadlineAt.Format("02.01"))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("del_toggle:%d", d.ID)),
		))
	}

	nSelected := 0
	for _, v := range selected {
		if v {
			nSelected++
		}
	}

	if nSelected == len(deadlines) {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("☑️ Deselect all", "del_none"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("☑️ Select all", "del_all"),
		))
	}

	deleteLabel := fmt.Sprintf("🗑 Delete (%d)", nSelected)
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(deleteLabel, "del_execute"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func HandleTextMessage(bot *tgbotapi.BotAPI, db *DB, msg *tgbotapi.Message, userID int64) {
	state := getState(userID)
	text := strings.TrimSpace(msg.Text)

	switch state.Step {
	case StepAddText:
		if text == "" {
			send(bot, msg.Chat.ID, "❌ Description can't be empty.")
			return
		}
		now := time.Now()
		state.TempText = text
		state.Step = StepAddDate
		state.CalYear = now.Year()
		state.CalMonth = now.Month()
		setState(userID, state)
		sendKb(bot, msg.Chat.ID, "📅 Pick a date:", calendarKeyboard(state.CalYear, state.CalMonth))

	case StepUpdateText:
		if text == "" {
			send(bot, msg.Chat.ID, "❌ Text can't be empty.")
			return
		}
		d, err := db.GetDeadlineByID(state.DeadlineID, userID)
		if err != nil {
			send(bot, msg.Chat.ID, "❌ "+err.Error())
			resetState(userID)
			return
		}
		if err := db.UpdateDeadline(state.DeadlineID, userID, text, d.DeadlineAt); err != nil {
			send(bot, msg.Chat.ID, "❌ "+err.Error())
		} else {
			send(bot, msg.Chat.ID, fmt.Sprintf("✅ Updated: *%s* - %s", text, d.DeadlineAt.Format("02.01.2006")))
		}
		resetState(userID)
	}
}

func HandleCallback(bot *tgbotapi.BotAPI, db *DB, cb *tgbotapi.CallbackQuery) {
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	data := cb.Data

	bot.Request(tgbotapi.NewCallback(cb.ID, ""))

	state := getState(userID)

	switch {
	case data == "cal_ignore":

	case strings.HasPrefix(data, "cal_day:"):
		date, err := parseCalDay(data)
		if err != nil {
			return
		}
		deleteMsg(bot, chatID, msgID)

		switch state.Step {
		case StepAddDate:
			state.TempDate = date
			state.Step = StepAddTime
			setState(userID, state)
			sendKb(bot, chatID, fmt.Sprintf("🕐 *%s* — pick a time:", date.Format("02.01.2006")), timeKeyboard())
		case StepUpdateDate:
			state.TempDate = date
			state.Step = StepUpdateTime
			setState(userID, state)
			sendKb(bot, chatID, fmt.Sprintf("🕐 *%s* — pick a time:", date.Format("02.01.2006")), timeKeyboard())
		}

	case strings.HasPrefix(data, "time_hm:"), data == "time_skip":
		var deadline time.Time
		if data == "time_skip" {
			deadline = time.Date(state.TempDate.Year(), state.TempDate.Month(), state.TempDate.Day(), 23, 59, 0, 0, time.Local)
		} else {
			h, m, err := parseTimeHM(data)
			if err != nil {
				return
			}
			deadline = time.Date(state.TempDate.Year(), state.TempDate.Month(), state.TempDate.Day(), h, m, 0, 0, time.Local)
		}
		deleteMsg(bot, chatID, msgID)

		switch state.Step {
		case StepAddTime:
			if err := db.AddDeadline(userID, state.TempText, deadline); err != nil {
				send(bot, chatID, "❌ Failed to save deadline.")
			} else {
				send(bot, chatID, fmt.Sprintf("✅ Added: *%s* — %s", state.TempText, deadline.Format("02.01.2006 15:04")))
			}
			resetState(userID)
		case StepUpdateTime:
			d, err := db.GetDeadlineByID(state.DeadlineID, userID)
			if err != nil {
				send(bot, chatID, "❌ "+err.Error())
				resetState(userID)
				return
			}
			if err := db.UpdateDeadline(state.DeadlineID, userID, d.Text, deadline); err != nil {
				send(bot, chatID, "❌ "+err.Error())
			} else {
				send(bot, chatID, fmt.Sprintf("✅ Updated: *%s* — %s", d.Text, deadline.Format("02.01.2006 15:04")))
			}
			resetState(userID)
		}

	case strings.HasPrefix(data, "cal_nav:"):
		year, month, err := parseCalNav(data)
		if err != nil {
			return
		}
		state.CalYear = year
		state.CalMonth = month
		setState(userID, state)
		bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, calendarKeyboard(year, month)))

	case strings.HasPrefix(data, "upd_select:"):
		var id int64
		fmt.Sscanf(strings.TrimPrefix(data, "upd_select:"), "%d", &id)
		deleteMsg(bot, chatID, msgID)

		state.Step = StepUpdateField
		state.DeadlineID = id
		setState(userID, state)

		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ Change text", fmt.Sprintf("upd_field:text:%d", id)),
				tgbotapi.NewInlineKeyboardButtonData("📅 Change date", fmt.Sprintf("upd_field:date:%d", id)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
			),
		)
		sendKb(bot, chatID, "What do you want to change?", kb)

	case strings.HasPrefix(data, "upd_field:"):
		parts := strings.SplitN(data, ":", 3)
		if len(parts) != 3 {
			return
		}
		var id int64
		fmt.Sscanf(parts[2], "%d", &id)
		deleteMsg(bot, chatID, msgID)

		switch parts[1] {
		case "text":
			state.Step = StepUpdateText
			state.DeadlineID = id
			setState(userID, state)
			send(bot, chatID, "✏️ Send the new description:")

		case "date":
			now := time.Now()
			state.Step = StepUpdateDate
			state.DeadlineID = id
			state.CalYear = now.Year()
			state.CalMonth = now.Month()
			setState(userID, state)
			sendKb(bot, chatID, "📅 Pick a new date:", calendarKeyboard(state.CalYear, state.CalMonth))
		}

	case strings.HasPrefix(data, "del_toggle:"):
		var id int64
		fmt.Sscanf(strings.TrimPrefix(data, "del_toggle:"), "%d", &id)
		if state.SelectedIDs == nil {
			state.SelectedIDs = make(map[int64]bool)
		}
		state.SelectedIDs[id] = !state.SelectedIDs[id]
		setState(userID, state)
		deadlines, _ := db.ListDeadlines(userID)
		bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, deleteKeyboard(deadlines, state.SelectedIDs)))

	case data == "del_all":
		deadlines, _ := db.ListDeadlines(userID)
		if state.SelectedIDs == nil {
			state.SelectedIDs = make(map[int64]bool)
		}
		for _, d := range deadlines {
			state.SelectedIDs[d.ID] = true
		}
		setState(userID, state)
		bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, deleteKeyboard(deadlines, state.SelectedIDs)))

	case data == "del_none":
		deadlines, _ := db.ListDeadlines(userID)
		state.SelectedIDs = make(map[int64]bool)
		setState(userID, state)
		bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, deleteKeyboard(deadlines, state.SelectedIDs)))

	case data == "del_execute":
		nSelected := 0
		for _, v := range state.SelectedIDs {
			if v {
				nSelected++
			}
		}
		if nSelected == 0 {
			bot.Request(tgbotapi.NewCallbackWithAlert(cb.ID, "Select at least one deadline."))
			return
		}
		var ids []int64
		for id, selected := range state.SelectedIDs {
			if selected {
				ids = append(ids, id)
			}
		}
		deleteMsg(bot, chatID, msgID)
		n, err := db.DeleteByIDs(userID, ids)
		if err != nil {
			send(bot, chatID, "❌ Could not delete deadlines.")
		} else {
			send(bot, chatID, fmt.Sprintf("🗑 Deleted %d deadline(s).", n))
		}
		resetState(userID)

	case data == "cancel":
		deleteMsg(bot, chatID, msgID)
		resetState(userID)
		send(bot, chatID, "Cancelled.")
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
