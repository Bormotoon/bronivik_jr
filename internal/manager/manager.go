package manager

import (
	"context"
	"fmt"
	"log"

	"bronivik/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleManagerAction(update tgbotapi.Update) {
	callback := update.CallbackQuery
	if callback == nil {
		return
	}

	data := callback.Data
	var bookingID int64
	var action string

	if _, err := fmt.Sscanf(data, "confirm_%d", &bookingID); err == nil {
		action = "confirm"
	} else if _, err := fmt.Sscanf(data, "reject_%d", &bookingID); err == nil {
		action = "reject"
	} else if _, err := fmt.Sscanf(data, "reschedule_%d", &bookingID); err == nil {
		action = "reschedule"
	} else {
		return
	}

	booking, err := b.db.GetBooking(context.Background(), bookingID)
	if err != nil {
		log.Printf("Error getting booking: %v", err)
		return
	}

	switch action {
	case "confirm":
		b.confirmBooking(booking, callback.Message.Chat.ID)
	case "reject":
		b.rejectBooking(booking, callback.Message.Chat.ID)
	case "reschedule":
		b.rescheduleBooking(booking, callback.Message.Chat.ID)
	}

	// Обновляем сообщение у менеджера
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID,
		fmt.Sprintf("✅ Заявка #%d обработана\nДействие: %s", bookingID, action))
	b.bot.Send(editMsg)
}

func (b *Bot) confirmBooking(booking models.Booking, managerChatID int64) {
	err := b.db.UpdateBookingStatus(context.Background(), booking.ID, "confirmed")
	if err != nil {
		log.Printf("Error confirming booking: %v", err)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("✅ Ваша заявка на %s подтверждена! Ждем вас %s.",
			booking.ItemName, booking.Date.Format("02.01.2006")))
	b.bot.Send(userMsg)

	// Уведомляем менеджера
	managerMsg := tgbotapi.NewMessage(managerChatID, "✅ Бронирование подтверждено")
	b.bot.Send(managerMsg)
}

func (b *Bot) rejectBooking(booking models.Booking, managerChatID int64) {
	err := b.db.UpdateBookingStatus(context.Background(), booking.ID, "cancelled")
	if err != nil {
		log.Printf("Error rejecting booking: %v", err)
		return
	}

	// Уведомляем пользователя
	userMsg := tgbotapi.NewMessage(booking.UserID,
		"❌ К сожалению, ваша заявка была отклонена менеджером.")
	b.bot.Send(userMsg)

	managerMsg := tgbotapi.NewMessage(managerChatID, "❌ Бронирование отменено")
	b.bot.Send(managerMsg)
}

func (b *Bot) rescheduleBooking(booking models.Booking, managerChatID int64) {
	// Отправляем пользователю сообщение с предложением выбрать другую дату
	userMsg := tgbotapi.NewMessage(booking.UserID,
		fmt.Sprintf("🔄 Менеджер предложил выбрать другую дату для %s. Пожалуйста, создайте новую заявку.",
			booking.ItemName))

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Создать заявку"),
		),
	)
	userMsg.ReplyMarkup = keyboard

	b.bot.Send(userMsg)

	// Обновляем статус текущей заявки
	err := b.db.UpdateBookingStatus(context.Background(), booking.ID, "rescheduled")
	if err != nil {
		log.Printf("Error updating booking status: %v", err)
	}

	managerMsg := tgbotapi.NewMessage(managerChatID, "🔄 Пользователю предложено выбрать другую дату")
	b.bot.Send(managerMsg)
}
