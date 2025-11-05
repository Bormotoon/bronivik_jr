package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bronivik/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleManagerCommand обработка команд менеджера
func (b *Bot) handleManagerCommand(update tgbotapi.Update) bool {
	if !b.isManager(update.Message.From.ID) {
		return false
	}

	userID := update.Message.From.ID
	text := update.Message.Text

	switch {
	case text == "👨‍💼 Все заявки":
		b.showManagerBookings(update)

	case text == "➕ Создать заявку (Менеджер)":
		b.startManagerBooking(update)

	case text == "/manager_export_week":
		b.handleExportWeek(update)

	case strings.HasPrefix(text, "/manager_export_range"):
		b.handleExportRange(update)

		// секретная команда, доступная менеджерам, но не отображаемся у них в меню
	case text == "/stats" && b.isManager(userID):
		b.getUserStats(update)

	case text == "💾 Экспорт недели":
		b.handleExportWeek(update)

	case text == "/manager_create_booking":
		b.startManagerBooking(update)

	case text == "/manager_bookings":
		b.showManagerBookings(update)

	case text == "/manager_availability":
		b.showManagerAvailability(update)

	case text == "/manager_export_week":
		b.handleExportWeek(update)

	case strings.HasPrefix(text, "/manager_export_range"):
		b.handleExportRange(update)

	case strings.HasPrefix(text, "/manager_booking_"):
		// Просмотр конкретной заявки
		parts := strings.Split(text, "_")
		if len(parts) >= 3 {
			bookingID, err := strconv.ParseInt(parts[2], 10, 64)
			if err == nil {
				b.showManagerBookingDetail(update, bookingID)
			}
		}

	case text == "🔄 Синхронизировать пользователей (Google Sheets)":
		b.SyncUsersToSheets()
		b.sendMessage(update.Message.Chat.ID, "✅ Пользователи синхронизированы с Google Таблицей")

	case text == "🔄 Синхронизировать бронирования (Google Sheets)":
		b.SyncBookingsToSheets()
		b.sendMessage(update.Message.Chat.ID, "✅ Бронирования синхронизированы с Google Таблицей")

	case text == "📅 Синхронизировать расписание (Google Sheets)":
		b.SyncScheduleToSheets()
		b.sendMessage(update.Message.Chat.ID, "✅ Расписание синхронизировано с Google Таблицей")
	}

	return false
}

// startManagerBooking начало создания заявки менеджером
func (b *Bot) startManagerBooking(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		"📋 Создание заявки от имени клиента\n\nВведите ID пользователя Telegram:")

	b.setUserState(update.Message.From.ID, "manager_waiting_user_id", map[string]interface{}{
		"is_manager_booking": true,
	})
	b.bot.Send(msg)
}

// showManagerBookings показывает все заявки менеджеру
func (b *Bot) showManagerBookings(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	// Получаем все заявки за последние 30 дней
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 30)

	bookings, err := b.db.GetBookingsByDateRange(context.Background(), startDate, endDate)
	if err != nil {
		log.Printf("Error getting bookings: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при получении заявок")
		return
	}

	var message strings.Builder
	message.WriteString("📊 Все заявки:\n\n")

	for _, booking := range bookings {
		statusEmoji := "⏳"
		switch booking.Status {
		case "confirmed":
			statusEmoji = "✅"
		case "cancelled":
			statusEmoji = "❌"
		case "changed":
			statusEmoji = "🔄"
		case "completed":
			statusEmoji = "🏁"
		}

		message.WriteString(fmt.Sprintf("%s Заявка #%d\n", statusEmoji, booking.ID))
		message.WriteString(fmt.Sprintf("   👤 %s\n", booking.UserName))
		message.WriteString(fmt.Sprintf("   🏢 %s\n", booking.ItemName))
		message.WriteString(fmt.Sprintf("   📅 %s\n", booking.Date.Format("02.01.2006")))
		message.WriteString(fmt.Sprintf("   📱 %s\n", booking.Phone))
		message.WriteString(fmt.Sprintf("   🔗 /manager_booking_%d\n\n", booking.ID))
	}

	if len(bookings) == 0 {
		message.WriteString("Заявок не найдено")
	}

	b.sendMessage(update.Message.Chat.ID, message.String())
}

// showManagerBookingDetail показывает детали заявки менеджеру
func (b *Bot) showManagerBookingDetail(update tgbotapi.Update, bookingID int64) {
	// ПРОВЕРКА НА NIL - чтобы избежать паники
	if update.Message == nil {
		log.Printf("Error: update.Message is nil in showManagerBookingDetail")
		return
	}

	booking, err := b.db.GetBooking(context.Background(), bookingID)
	if err != nil {
		b.sendMessage(update.Message.Chat.ID, "Заявка не найдена")
		return
	}

	statusText := map[string]string{
		"pending":   "⏳ Ожидает подтверждения",
		"confirmed": "✅ Подтверждена",
		"cancelled": "❌ Отменена",
		"changed":   "🔄 Изменена",
		"completed": "🏁 Завершена",
	}

	message := fmt.Sprintf(`📋 Заявка #%d

👤 Клиент: %s
📱 Телефон: %s
🏢 Позиция: %s
📅 Дата: %s
📊 Статус: %s
🕐 Создана: %s
✏️ Обновлена: %s`,
		booking.ID,
		booking.UserName,
		booking.Phone,
		booking.ItemName,
		booking.Date.Format("02.01.2006"),
		statusText[booking.Status],
		booking.CreatedAt.Format("02.01.2006 15:04"),
		booking.UpdatedAt.Format("02.01.2006 15:04"),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)

	// Создаем инлайн-клавиатуру для управления заявкой
	var rows [][]tgbotapi.InlineKeyboardButton

	if booking.Status == "pending" || booking.Status == "changed" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("confirm_%d", booking.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", booking.ID)),
		))
	}

	if booking.Status == "confirmed" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Вернуть в работу", fmt.Sprintf("reopen_%d", booking.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🏁 Завершить", fmt.Sprintf("complete_%d", booking.ID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить аппарат", fmt.Sprintf("change_item_%d", booking.ID)),
		tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить", fmt.Sprintf("tel:%s", booking.Phone)),
	))

	if len(rows) > 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
		msg.ReplyMarkup = &keyboard
	}

	b.bot.Send(msg)
}

// handleManagerAction обработка действий менеджера с заявками
func (b *Bot) handleManagerAction(update tgbotapi.Update) {
	callback := update.CallbackQuery
	if callback == nil {
		return
	}

	data := callback.Data
	var bookingID int64
	var action string

	// Обрабатываем все возможные действия
	actions := []string{"confirm_", "reject_", "reschedule_", "change_item_", "reopen_", "complete_"}
	for _, act := range actions {
		if _, err := fmt.Sscanf(data, act+"%d", &bookingID); err == nil {
			action = act
			break
		}
	}

	if action == "" {
		return
	}

	booking, err := b.db.GetBooking(context.Background(), bookingID)
	if err != nil {
		log.Printf("Error getting booking: %v", err)
		return
	}

	switch action {
	case "confirm_":
		b.confirmBooking(booking, callback.Message.Chat.ID)
	case "reject_":
		b.rejectBooking(booking, callback.Message.Chat.ID)
	case "reschedule_":
		b.rescheduleBooking(booking, callback.Message.Chat.ID)
	case "change_item_":
		b.startChangeItem(booking, callback.Message.Chat.ID)
	case "reopen_":
		b.reopenBooking(booking, callback.Message.Chat.ID)
	case "complete_":
		b.completeBooking(booking, callback.Message.Chat.ID)
	}

	// Обновляем сообщение у менеджера
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID,
		fmt.Sprintf("✅ Заявка #%d обработана\nДействие: %s", bookingID, action))
	b.bot.Send(editMsg)
}

// startChangeItem начало изменения аппарата в заявке
func (b *Bot) startChangeItem(booking *models.Booking, managerChatID int64) {
	msg := tgbotapi.NewMessage(managerChatID,
		"Выберите новый аппарат для заявки #"+strconv.FormatInt(booking.ID, 10)+":")

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	for _, item := range b.items {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(item.Name,
				fmt.Sprintf("change_to_%d_%d", booking.ID, item.ID)),
		)
		keyboardRows = append(keyboardRows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	msg.ReplyMarkup = &keyboard

	b.bot.Send(msg)
}

// handleChangeItem обработка выбора нового аппарата С ПРОВЕРКОЙ ДОСТУПНОСТИ
func (b *Bot) handleChangeItem(update tgbotapi.Update) {
	callback := update.CallbackQuery
	if callback == nil {
		return
	}

	data := callback.Data
	var bookingID, itemID int64

	if _, err := fmt.Sscanf(data, "change_to_%d_%d", &bookingID, &itemID); err != nil {
		return
	}

	// Находим выбранный аппарат
	var selectedItem models.Item
	for _, item := range b.items {
		if item.ID == itemID {
			selectedItem = item
			break
		}
	}

	if selectedItem.ID == 0 {
		b.sendMessage(callback.Message.Chat.ID, "Аппарат не найден")
		return
	}

	// ПРОВЕРЯЕМ ДОСТУПНОСТЬ нового аппарата на дату заявки
	booking, available, err := b.db.GetBookingWithAvailability(context.Background(), bookingID, selectedItem.ID)
	if err != nil {
		log.Printf("Error checking availability: %v", err)
		b.sendMessage(callback.Message.Chat.ID, "Ошибка при проверке доступности")
		return
	}

	if !available {
		b.sendMessage(callback.Message.Chat.ID,
			fmt.Sprintf("❌ Аппарат '%s' недоступен на дату %s. Выберите другой аппарат.",
				selectedItem.Name, booking.Date.Format("02.01.2006")))
		return
	}

	// Обновляем заявку
	err = b.db.UpdateBookingItem(context.Background(), bookingID, selectedItem.ID, selectedItem.Name)
	if err != nil {
		log.Printf("Error updating booking item: %v", err)
		b.sendMessage(callback.Message.Chat.ID, "Ошибка при обновлении заявки")
		return
	}

	// Обновляем статус
	err = b.db.UpdateBookingStatus(context.Background(), bookingID, "changed")
	if err != nil {
		log.Printf("Error updating booking status: %v", err)
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("🔄 В вашей заявке #%d изменен аппарат на: %s", bookingID, selectedItem.Name))
	b.bot.Send(userMsg)

	b.sendMessage(callback.Message.Chat.ID, "✅ Аппарат успешно изменен")

	// ВМЕСТО ВЫЗОВА showManagerBookingDetail, который требует Message, используем sendManagerBookingDetail
	updatedBooking, err := b.db.GetBooking(context.Background(), bookingID)
	if err != nil {
		log.Printf("Error getting updated booking: %v", err)
		return
	}

	// Отправляем обновленные детали заявки
	b.sendManagerBookingDetail(callback.Message.Chat.ID, updatedBooking)
}

// sendManagerBookingDetail отправляет детали заявки в указанный чат (без использования update)
func (b *Bot) sendManagerBookingDetail(chatID int64, booking *models.Booking) {
	statusText := map[string]string{
		"pending":   "⏳ Ожидает подтверждения",
		"confirmed": "✅ Подтверждена",
		"cancelled": "❌ Отменена",
		"changed":   "🔄 Изменена",
		"completed": "🏁 Завершена",
	}

	message := fmt.Sprintf(`📋 Заявка #%d

👤 Клиент: %s
📱 Телефон: %s
🏢 Позиция: %s
📅 Дата: %s
📊 Статус: %s
🕐 Создана: %s
✏️ Обновлена: %s`,
		booking.ID,
		booking.UserName,
		booking.Phone,
		booking.ItemName,
		booking.Date.Format("02.01.2006"),
		statusText[booking.Status],
		booking.CreatedAt.Format("02.01.2006 15:04"),
		booking.UpdatedAt.Format("02.01.2006 15:04"),
	)

	msg := tgbotapi.NewMessage(chatID, message)

	// Создаем инлайн-клавиатуру для управления заявкой
	var rows [][]tgbotapi.InlineKeyboardButton

	if booking.Status == "pending" || booking.Status == "changed" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("confirm_%d", booking.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", booking.ID)),
		))
	}

	if booking.Status == "confirmed" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Вернуть в работу", fmt.Sprintf("reopen_%d", booking.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🏁 Завершить", fmt.Sprintf("complete_%d", booking.ID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить аппарат", fmt.Sprintf("change_item_%d", booking.ID)),
		tgbotapi.NewInlineKeyboardButtonData("📞 Позвонить", fmt.Sprintf("tel:%s", booking.Phone)),
	))

	if len(rows) > 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
		msg.ReplyMarkup = &keyboard
	}

	b.bot.Send(msg)
}

// reopenBooking возврат заявки в работу
func (b *Bot) reopenBooking(booking *models.Booking, managerChatID int64) {
	err := b.db.UpdateBookingStatus(context.Background(), booking.ID, "pending")
	if err != nil {
		log.Printf("Error reopening booking: %v", err)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("🔄 Ваша заявка #%d возвращена в работу. Ожидайте подтверждения.", booking.ID))
	b.bot.Send(userMsg)

	managerMsg := tgbotapi.NewMessage(managerChatID, "✅ Заявка возвращена в работу")
	b.bot.Send(managerMsg)
}

// completeBooking завершение заявки
func (b *Bot) completeBooking(booking *models.Booking, managerChatID int64) {
	err := b.db.UpdateBookingStatus(context.Background(), booking.ID, "completed")
	if err != nil {
		log.Printf("Error completing booking: %v", err)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("🏁 Ваша заявка #%d завершена. Спасибо за использование наших услуг!", booking.ID))
	b.bot.Send(userMsg)

	managerMsg := tgbotapi.NewMessage(managerChatID, "✅ Заявка завершена")
	b.bot.Send(managerMsg)
}

// showManagerAvailability показывает доступность аппаратов на неделю
func (b *Bot) showManagerAvailability(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	startDate := time.Now()
	var message strings.Builder
	message.WriteString("📊 Доступность аппаратов на ближайшие 7 дней:\n\n")

	for _, item := range b.items {
		message.WriteString(fmt.Sprintf("🏢 %s (всего: %d):\n", item.Name, item.TotalQuantity))

		availability, err := b.db.GetAvailabilityForPeriod(context.Background(), item.ID, startDate, 7)
		if err != nil {
			log.Printf("Error getting availability: %v", err)
			message.WriteString("   Ошибка получения данных\n")
			continue
		}

		for _, avail := range availability {
			status := fmt.Sprintf("✅ Свободно (%d/%d)", avail.Available, item.TotalQuantity)
			if avail.Available == 0 {
				status = fmt.Sprintf("❌ Занято (%d/%d)", avail.Booked, item.TotalQuantity)
			} else if avail.Available < item.TotalQuantity {
				status = fmt.Sprintf("⚠️  Частично занято (%d/%d)", avail.Booked, item.TotalQuantity)
			}

			message.WriteString(fmt.Sprintf("   %s: %s\n",
				avail.Date.Format("02.01"), status))
		}
		message.WriteString("\n")
	}

	// Добавляем команды для экспорта
	message.WriteString("💾 Команды для экспорта:\n")
	message.WriteString("/manager_export_week - экспорт текущей недели\n")
	message.WriteString("/manager_export_range 2024-01-01 2024-01-31 - экспорт за период\n")

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message.String())
	b.bot.Send(msg)
}

// handleExportWeek экспорт данных за текущую неделю
func (b *Bot) handleExportWeek(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, 6) // +6 дней = неделя

	filePath, err := b.exportToExcel(startDate, endDate)
	if err != nil {
		log.Printf("Error exporting to Excel: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при создании файла экспорта")
		return
	}

	// ОТПРАВКА ФАЙЛА
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при открытии файла")
		return
	}
	defer file.Close()

	fileReader := tgbotapi.FileReader{
		Name:   filepath.Base(filePath),
		Reader: file,
	}

	doc := tgbotapi.NewDocument(update.Message.Chat.ID, fileReader)
	doc.Caption = fmt.Sprintf("📊 Экспорт данных с %s по %s",
		startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))

	_, err = b.bot.Send(doc)
	if err != nil {
		log.Printf("Error sending document: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при отправке файла")
		return
	}

	b.sendMessage(update.Message.Chat.ID, "✅ Файл экспорта успешно отправлен")
}

// handleExportRange экспорт данных за указанный период
func (b *Bot) handleExportRange(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) != 3 {
		b.sendMessage(update.Message.Chat.ID,
			"Использование: /manager_export_range ГГГГ-ММ-ДД ГГГГ-ММ-ДД\nПример: /manager_export_range 2024-01-01 2024-01-31")
		return
	}

	startDate, err1 := time.Parse("2006-01-02", parts[1])
	endDate, err2 := time.Parse("2006-01-02", parts[2])

	if err1 != nil || err2 != nil {
		b.sendMessage(update.Message.Chat.ID, "Неверный формат даты. Используйте: ГГГГ-ММ-ДД")
		return
	}

	if startDate.After(endDate) {
		b.sendMessage(update.Message.Chat.ID, "Начальная дата не может быть позже конечной")
		return
	}

	filePath, err := b.exportToExcel(startDate, endDate)
	if err != nil {
		log.Printf("Error exporting to Excel: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при создании файла экспорта")
		return
	}

	// ОТПРАВКА ФАЙЛА
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при открытии файла")
		return
	}
	defer file.Close()

	fileReader := tgbotapi.FileReader{
		Name:   filepath.Base(filePath),
		Reader: file,
	}

	doc := tgbotapi.NewDocument(update.Message.Chat.ID, fileReader)
	doc.Caption = fmt.Sprintf("📊 Экспорт данных с %s по %s",
		startDate.Format("02.01.2006"), endDate.Format("02.01.2006"))

	_, err = b.bot.Send(doc)
	if err != nil {
		log.Printf("Error sending document: %v", err)
		b.sendMessage(update.Message.Chat.ID, "Ошибка при отправке файла")
		return
	}

	b.sendMessage(update.Message.Chat.ID, "✅ Файл экспорта успешно отправлен")
}

// SyncScheduleToSheets синхронизирует расписание в формате таблицы с Google Sheets
func (b *Bot) SyncScheduleToSheets() {
	if b.sheetsService == nil {
		return
	}

	// Определяем период (например, текущая неделя)
	startDate := time.Now().Truncate(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, 6) // +6 дней = неделя

	// Получаем данные о бронированиях
	dailyBookings, err := b.db.GetDailyBookings(context.Background(), startDate, endDate)
	if err != nil {
		log.Printf("Failed to get daily bookings for schedule sync: %v", err)
		return
	}

	// Конвертируем модели в google-модели
	googleDailyBookings := make(map[string][]models.Booking)
	for date, bookings := range dailyBookings {
		var googleBookings []models.Booking
		for _, booking := range bookings {
			googleBookings = append(googleBookings, models.Booking{
				ID:        booking.ID,
				UserID:    booking.UserID,
				ItemID:    booking.ItemID,
				Date:      booking.Date,
				Status:    booking.Status,
				UserName:  booking.UserName,
				Phone:     booking.Phone,
				ItemName:  booking.ItemName,
				CreatedAt: booking.CreatedAt,
				UpdatedAt: booking.UpdatedAt,
			})
		}
		googleDailyBookings[date] = googleBookings
	}

	// Конвертируем items
	var googleItems []models.Item
	for _, item := range b.items {
		googleItems = append(googleItems, models.Item{
			ID:            item.ID,
			Name:          item.Name,
			TotalQuantity: item.TotalQuantity,
		})
	}

	// Обновляем расписание в Google Sheets
	err = b.sheetsService.UpdateScheduleSheet(startDate, endDate, googleDailyBookings, googleItems)
	if err != nil {
		log.Printf("Failed to sync schedule to Google Sheets: %v", err)
	} else {
		log.Println("Schedule successfully synced to Google Sheets")
	}
}
