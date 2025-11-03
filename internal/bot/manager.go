package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"bronivik/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleManagerCommand обработка команд менеджера
func (b *Bot) handleManagerCommand(update tgbotapi.Update) {
	if !b.isManager(update.Message.From.ID) {
		return
	}

	text := update.Message.Text

	switch {
	case text == "/manager_create_booking":
		b.startManagerBooking(update)
	case text == "/manager_bookings":
		b.showManagerBookings(update)
	case strings.HasPrefix(text, "/manager_booking_"):
		// Просмотр конкретной заявки
		parts := strings.Split(text, "_")
		if len(parts) >= 3 {
			bookingID, err := strconv.ParseInt(parts[2], 10, 64)
			if err == nil {
				b.showManagerBookingDetail(update, bookingID)
			}
		}
	}
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

// handleChangeItem обработка выбора нового аппарата
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

	// Обновляем заявку
	err := b.db.UpdateBookingItem(context.Background(), bookingID, selectedItem.ID, selectedItem.Name)
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
	booking, _ := b.db.GetBooking(context.Background(), bookingID)
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("🔄 В вашей заявке #%d изменен аппарат на: %s", bookingID, selectedItem.Name))
	b.bot.Send(userMsg)

	b.sendMessage(callback.Message.Chat.ID, "✅ Аппарат успешно изменен")
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
