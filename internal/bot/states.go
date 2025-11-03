package bot

//
// import (
// 	"fmt"
// 	"time"
//
// 	"bronivik/internal/models"
// 	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
// )
//
// // const (
// // 	StateMainMenu     = "main_menu"
// // 	StateSelectItem   = "select_item"
// // 	StateSelectDate   = "select_date"
// // 	StateViewSchedule = "view_schedule"
// // 	StatePersonalData = "personal_data"
// // 	StatePhoneNumber  = "phone_number"
// // 	StateConfirmation = "confirmation"
// // )
//
// func (b *Bot) handleMainMenu(update tgbotapi.Update) {
// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
// 		"Добро пожаловать! Выберите действие:")
//
// 	keyboard := tgbotapi.NewReplyKeyboard(
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("📅 Посмотреть расписание"),
// 			tgbotapi.NewKeyboardButton("💼 Доступные позиции"),
// 		),
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("📋 Создать заявку"),
// 		),
// 	)
// 	msg.ReplyMarkup = keyboard
//
// 	b.setUserState(update.Message.From.ID, StateMainMenu, nil)
// 	b.bot.Send(msg)
// }
//
// func (b *Bot) handleSelectItem(update tgbotapi.Update) {
// 	items := b.config.Items
// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
// 		"Выберите позицию для бронирования:")
//
// 	var keyboardRows [][]tgbotapi.KeyboardButton
// 	for _, item := range items {
// 		row := tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton(fmt.Sprintf("🏢 %s", item.Name)),
// 		)
// 		keyboardRows = append(keyboardRows, row)
// 	}
//
// 	keyboardRows = append(keyboardRows, tgbotapi.NewKeyboardButtonRow(
// 		tgbotapi.NewKeyboardButton("⬅️ Назад"),
// 	))
//
// 	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboardRows...)
// 	b.setUserState(update.Message.From.ID, StateSelectItem, nil)
// 	b.bot.Send(msg)
// }
//
// func (b *Bot) handleViewSchedule(update tgbotapi.Update) {
// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
// 		"Выберите период для просмотра расписания:")
//
// 	keyboard := tgbotapi.NewReplyKeyboard(
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("📅 7 дней"),
// 			tgbotapi.NewKeyboardButton("🗓 Выбрать дату"),
// 		),
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("⬅️ Назад"),
// 		),
// 	)
// 	msg.ReplyMarkup = keyboard
//
// 	b.setUserState(update.Message.From.ID, StateViewSchedule, nil)
// 	b.bot.Send(msg)
// }
//
// func (b *Bot) handlePersonalData(update tgbotapi.Update, itemID int, date time.Time) {
// 	state := b.getUserState(update.Message.From.ID)
// 	if state == nil {
// 		state = &models.UserState{
// 			UserID:   update.Message.From.ID,
// 			TempData: make(map[string]interface{}),
// 		}
// 	}
//
// 	state.TempData["item_id"] = itemID
// 	state.TempData["date"] = date
// 	b.setUserState(update.Message.From.ID, StatePersonalData, state.TempData)
//
// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
// 		`Для оформления заявки необходимо ваше согласие на обработку персональных данных.
//
// Мы обязуемся использовать ваши данные исключительно для обработки заявки и связи с вами.`)
//
// 	keyboard := tgbotapi.NewReplyKeyboard(
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("✅ Даю согласие"),
// 		),
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("❌ Отмена"),
// 		),
// 	)
// 	msg.ReplyMarkup = keyboard
//
// 	b.bot.Send(msg)
// }
//
// func (b *Bot) handlePhoneRequest(update tgbotapi.Update) {
// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
// 		"Пожалуйста, предоставьте ваш номер телефона для связи:")
//
// 	keyboard := tgbotapi.NewReplyKeyboard(
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButtonContact("📱 Отправить номер телефона"),
// 		),
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("❌ Отмена"),
// 		),
// 	)
// 	msg.ReplyMarkup = keyboard
//
// 	b.setUserState(update.Message.From.ID, StatePhoneNumber, nil)
// 	b.bot.Send(msg)
// }
//
