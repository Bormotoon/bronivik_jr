package manager

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleExport(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	// Парсим даты из команды /export 2024-01-01 2024-01-31
	parts := strings.Fields(update.Message.Text)
	if len(parts) != 3 {
		b.sendMessage(update.Message.Chat.ID, "Использование: /export ГГГГ-ММ-ДД ГГГГ-ММ-ДД")
		return
	}

	startDate, err1 := time.Parse("2006-01-02", parts[1])
	endDate, err2 := time.Parse("2006-01-02", parts[2])

	if err1 != nil || err2 != nil {
		b.sendMessage(update.Message.Chat.ID, "Неверный формат даты. Используйте: ГГГГ-ММ-ДД")
		return
	}

	bookings, err := b.db.GetBookingsByDateRange(context.Background(), startDate, endDate)
	if err != nil {
		log.Printf("Error getting bookings: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при получении данных")
		return
	}

	filename, err := b.createCSVExport(bookings, startDate, endDate)
	if err != nil {
		log.Printf("Error creating CSV: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при создании файла")
		return
	}

	// Отправляем файл менеджеру
	doc := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, filename)
	b.bot.Send(doc)

	// Удаляем временный файл
	os.Remove(filename)
}

func (b *Bot) createCSVExport(bookings []models.Booking, startDate, endDate time.Time) (string, error) {
	filename := filepath.Join(b.config.Export.Path,
		fmt.Sprintf("bookings_%s_%s.csv",
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02")))

	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовки
	headers := []string{"ID", "Позиция", "Дата", "Клиент", "Телефон", "Статус", "Создана"}
	writer.Write(headers)

	// Данные
	for _, booking := range bookings {
		record := []string{
			fmt.Sprintf("%d", booking.ID),
			booking.ItemName,
			booking.Date.Format("02.01.2006"),
			booking.UserName,
			booking.Phone,
			booking.Status,
			booking.CreatedAt.Format("02.01.2006 15:04"),
		}
		writer.Write(record)
	}

	return filename, nil
}
