package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"bronivik/internal/models"
	"github.com/xuri/excelize/v2"
)

// exportToExcel создает Excel файл с данными о бронированиях
func (b *Bot) exportToExcel(startDate, endDate time.Time) (string, error) {
	// Создаем папку для экспорта, если не существует
	if err := os.MkdirAll(b.config.Exports.Path, 0755); err != nil {
		return "", fmt.Errorf("error creating export directory: %v", err)
	}

	// Получаем данные из БД
	dailyBookings, err := b.db.GetDailyBookings(context.Background(), startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("error getting bookings: %v", err)
	}

	items := b.items

	// Создаем новый Excel файл
	f := excelize.NewFile()

	// Создаем лист с данными
	index, err := f.NewSheet("Бронирования")
	if err != nil {
		return "", fmt.Errorf("error creating sheet: %v", err)
	}
	f.SetActiveSheet(index)

	// Устанавливаем заголовок периода
	f.SetCellValue("Бронирования", "A1", fmt.Sprintf("Период: %s - %s",
		startDate.Format("02.01.2006"), endDate.Format("02.01.2006")))

	// Заголовки - даты (начинаем с строки 2)
	col := 2 // Начинаем с колонки B
	currentDate := startDate
	dateHeaders := make(map[string]int) // для быстрого доступа к колонкам по дате

	for !currentDate.After(endDate) {
		cell, _ := excelize.CoordinatesToCellName(col, 2) // строка 2 для заголовков дат
		dateStr := currentDate.Format("02.01")
		f.SetCellValue("Бронирования", cell, dateStr)
		dateHeaders[currentDate.Format("2006-01-02")] = col

		// Форматируем заголовки дат
		style, err := f.NewStyle(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DDEBF7"}, Pattern: 1},
			Font:      &excelize.Font{Bold: true},
			Alignment: &excelize.Alignment{Horizontal: "center"},
		})
		if err == nil {
			f.SetCellStyle("Бронирования", cell, cell, style)
		}

		col++
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Названия аппаратов в первом столбце (начинаем с строки 3)
	row := 3
	for _, item := range items {
		cell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue("Бронирования", cell, fmt.Sprintf("%s (%d)", item.Name, item.TotalQuantity))

		// Форматируем названия аппаратов
		style, err := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
			Font: &excelize.Font{Bold: true},
		})
		if err == nil {
			f.SetCellStyle("Бронирования", cell, cell, style)
		}

		row++
	}

	// Заполняем данные по бронированиям
	for dateKey, bookings := range dailyBookings {
		col, exists := dateHeaders[dateKey]
		if !exists {
			continue
		}

		// Группируем бронирования по аппаратам
		bookingsByItem := make(map[int64][]models.Booking)
		for _, booking := range bookings {
			bookingsByItem[booking.ItemID] = append(bookingsByItem[booking.ItemID], booking)
		}

		// Заполняем данные для каждого аппарата
		row := 3
		for _, item := range items {
			cell, _ := excelize.CoordinatesToCellName(col, row)

			itemBookings := bookingsByItem[item.ID]
			date, _ := time.Parse("2006-01-02", dateKey)

			// Проверяем доступность аппарата на эту дату
			available, err := b.db.CheckAvailability(context.Background(), item.ID, date)
			if err != nil {
				log.Printf("Error checking availability for export: %v", err)
				available = false
			}

			// Получаем количество занятых аппаратов
			bookedCount, err := b.db.GetBookedCount(context.Background(), item.ID, date)
			if err != nil {
				log.Printf("Error getting booked count: %v", err)
				bookedCount = 0
			}

			if len(itemBookings) > 0 {
				var cellValue string
				var hasUnconfirmed bool

				for _, booking := range itemBookings {
					status := "❓"
					if booking.Status == "confirmed" {
						status = "✅"
					} else if booking.Status == "pending" || booking.Status == "changed" {
						status = "⏳"
						hasUnconfirmed = true
					}

					cellValue += fmt.Sprintf("%s %s (%s)\n", status, booking.UserName, booking.Phone)
				}

				// Добавляем информацию о доступности
				cellValue += fmt.Sprintf("\nЗанято: %d/%d", bookedCount, item.TotalQuantity)

				f.SetCellValue("Бронирования", cell, cellValue)

				// ОПРЕДЕЛЯЕМ ЦВЕТ ПО НОВЫМ ПРАВИЛАМ:
				// Красный - если аппарат полностью занят (не доступен)
				// Желтый - если есть хотя бы одна неподтвержденная заявка
				// Зеленый - если есть свободные аппараты и все заявки подтверждены
				styleID, err := b.getCellStyleByAvailability(f, available, hasUnconfirmed, bookedCount, int(item.TotalQuantity))
				if err == nil {
					f.SetCellStyle("Бронирования", cell, cell, styleID)
				}
			} else {
				// Нет заявок - свободно
				cellValue := fmt.Sprintf("Свободно\n\nДоступно: %d/%d", item.TotalQuantity, item.TotalQuantity)
				f.SetCellValue("Бронирования", cell, cellValue)

				// Зеленый для свободных ячеек
				style, err := f.NewStyle(&excelize.Style{
					Fill: excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
					Alignment: &excelize.Alignment{
						Horizontal: "left",
						Vertical:   "top",
						WrapText:   true,
					},
				})
				if err == nil {
					f.SetCellStyle("Бронирования", cell, cell, style)
				}
			}

			row++
		}
	}

	// Настраиваем ширину колонок
	f.SetColWidth("Бронирования", "A", "A", 25) // Названия аппаратов
	for i := 'B'; i < 'Z'; i++ {
		f.SetColWidth("Бронирования", string(i), string(i), 20)
	}

	// Объединяем ячейку для заголовка периода
	lastCol := getLastColumn(len(dateHeaders) + 1)
	f.MergeCell("Бронирования", "A1", lastCol+"1")

	// Стиль для заголовка периода
	style, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err == nil {
		f.SetCellStyle("Бронирования", "A1", "A1", style)
	}

	// Удаляем стандартный лист "Sheet1"
	f.DeleteSheet("Sheet1")

	// Сохраняем файл
	fileName := fmt.Sprintf("export_%s_to_%s.xlsx",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	filePath := filepath.Join(b.config.Exports.Path, fileName)

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("error saving file: %v", err)
	}

	log.Printf("Excel file created: %s", filePath)
	return filePath, nil
}

// getCellStyleByAvailability возвращает стиль ячейки на основе доступности и статусов заявок
func (b *Bot) getCellStyleByAvailability(f *excelize.File, available bool, hasUnconfirmed bool, bookedCount int, totalQuantity int) (int, error) {
	var fillColor string

	// НОВАЯ ЛОГИКА ЦВЕТОВ:
	// 1. Красный - если аппарат полностью занят (не доступен)
	if !available {
		fillColor = "#FFC7CE" // Красный
	} else if hasUnconfirmed { // 2. Желтый - если есть хотя бы одна неподтвержденная заявка
		fillColor = "#FFEB9C" // Желтый
	} else { // 3. Зеленый - если есть свободные аппараты и все заявки подтверждены
		fillColor = "#C6EFCE" // Зеленый
	}

	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{fillColor}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
	})
	if err != nil {
		log.Printf("Error creating cell style: %v", err)
		return 0, err
	}

	return style, nil
}

// getLastColumn возвращает последнюю колонку для объединения ячеек
func getLastColumn(colCount int) string {
	// Базовые колонки A-Z
	if colCount <= 26 {
		return string('A' + colCount - 1)
	}

	// Для большего количества колонок (AA, AB, etc.)
	firstChar := string('A' + (colCount-1)/26 - 1)
	secondChar := string('A' + (colCount-1)%26)
	return firstChar + secondChar
}

// exportUsersToExcel создает Excel файл с данными пользователей
func (b *Bot) exportUsersToExcel(users []models.User) (string, error) {
	// Создаем папку для экспорта, если не существует
	if err := os.MkdirAll(b.config.Exports.Path, 0755); err != nil {
		return "", fmt.Errorf("error creating export directory: %v", err)
	}

	// Создаем новый Excel файл
	f := excelize.NewFile()

	// Создаем лист с пользователями
	index, err := f.NewSheet("Пользователи")
	if err != nil {
		return "", fmt.Errorf("error creating sheet: %v", err)
	}
	f.SetActiveSheet(index)

	// Заголовки
	headers := []string{"ID", "Telegram ID", "Username", "Имя", "Фамилия", "Телефон", "Менеджер", "Черный список", "Язык", "Последняя активность", "Дата регистрации"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Пользователи", cell, header)
		// f.SetCellStyle("Пользователи", cell, cell, f.SetCellStyle("Пользователи", cell, "bold")
	}

	// Данные пользователей
	for i, user := range users {
		row := i + 2
		f.SetCellValue("Пользователи", fmt.Sprintf("A%d", row), user.ID)
		f.SetCellValue("Пользователи", fmt.Sprintf("B%d", row), user.TelegramID)
		f.SetCellValue("Пользователи", fmt.Sprintf("C%d", row), user.Username)
		f.SetCellValue("Пользователи", fmt.Sprintf("D%d", row), user.FirstName)
		f.SetCellValue("Пользователи", fmt.Sprintf("E%d", row), user.LastName)
		f.SetCellValue("Пользователи", fmt.Sprintf("F%d", row), user.Phone)
		f.SetCellValue("Пользователи", fmt.Sprintf("G%d", row), boolToYesNo(user.IsManager))
		f.SetCellValue("Пользователи", fmt.Sprintf("H%d", row), boolToYesNo(user.IsBlacklisted))
		f.SetCellValue("Пользователи", fmt.Sprintf("I%d", row), user.LanguageCode)
		f.SetCellValue("Пользователи", fmt.Sprintf("J%d", row), user.LastActivity.Format("02.01.2006 15:04"))
		f.SetCellValue("Пользователи", fmt.Sprintf("K%d", row), user.CreatedAt.Format("02.01.2006 15:04"))
	}

	// Настраиваем ширину колонок
	f.SetColWidth("Пользователи", "A", "A", 10)
	f.SetColWidth("Пользователи", "B", "B", 15)
	f.SetColWidth("Пользователи", "C", "C", 20)
	f.SetColWidth("Пользователи", "D", "D", 15)
	f.SetColWidth("Пользователи", "E", "E", 15)
	f.SetColWidth("Пользователи", "F", "F", 15)
	f.SetColWidth("Пользователи", "G", "G", 10)
	f.SetColWidth("Пользователи", "H", "H", 12)
	f.SetColWidth("Пользователи", "I", "I", 10)
	f.SetColWidth("Пользователи", "J", "J", 20)
	f.SetColWidth("Пользователи", "K", "K", 20)

	// Удаляем стандартный лист
	f.DeleteSheet("Sheet1")

	// Сохраняем файл
	fileName := fmt.Sprintf("users_export_%s.xlsx", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(b.config.Exports.Path, fileName)

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("error saving file: %v", err)
	}

	log.Printf("Users Excel file created: %s", filePath)
	return filePath, nil
}

// boolToYesNo преобразует bool в "Да"/"Нет"
func boolToYesNo(b bool) string {
	if b {
		return "Да"
	}
	return "Нет"
}
