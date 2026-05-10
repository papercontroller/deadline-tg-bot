package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var monthNames = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func calendarKeyboard(year int, month time.Month) tgbotapi.InlineKeyboardMarkup {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := firstDay.AddDate(0, 1, -1).Day()
	offset := int((firstDay.Weekday() + 6) % 7)

	py, pm := shiftMonth(year, month, -1)
	ny, nm := shiftMonth(year, month, 1)

	var rows [][]tgbotapi.InlineKeyboardButton

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀", fmt.Sprintf("cal_nav:%d-%02d", py, pm)),
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %d", monthNames[month-1], year), "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("▶", fmt.Sprintf("cal_nav:%d-%02d", ny, nm)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Mo", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Tu", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("We", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Th", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Fr", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Sa", "cal_ignore"),
		tgbotapi.NewInlineKeyboardButtonData("Su", "cal_ignore"),
	))

	cells := make([]int, offset)
	for d := 1; d <= daysInMonth; d++ {
		cells = append(cells, d)
	}
	for len(cells)%7 != 0 {
		cells = append(cells, 0)
	}

	today := time.Now()
	todayMidnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	for i := 0; i < len(cells); i += 7 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 7; j++ {
			d := cells[i+j]
			if d == 0 {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(" ", "cal_ignore"))
				continue
			}
			date := time.Date(year, month, d, 0, 0, 0, 0, time.Local)
			label := strconv.Itoa(d)
			cbData := fmt.Sprintf("cal_day:%d-%02d-%02d", year, int(month), d)
			if date.Before(todayMidnight) {
				label = "·"
				cbData = "cal_ignore"
			}
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, cbData))
		}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func shiftMonth(year int, month time.Month, delta int) (int, time.Month) {
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.Local).AddDate(0, delta, 0)
	return t.Year(), t.Month()
}

func parseCalDay(data string) (time.Time, error) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid cal_day data")
	}
	t, err := time.ParseInLocation("2006-01-02", parts[1], time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 0, 0, time.Local), nil
}

func parseCalNav(data string) (int, time.Month, error) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid cal_nav data")
	}
	dateParts := strings.Split(parts[1], "-")
	if len(dateParts) != 2 {
		return 0, 0, fmt.Errorf("invalid cal_nav data")
	}
	year, err := strconv.Atoi(dateParts[0])
	if err != nil {
		return 0, 0, err
	}
	m, err := strconv.Atoi(dateParts[1])
	if err != nil {
		return 0, 0, err
	}
	return year, time.Month(m), nil
}
