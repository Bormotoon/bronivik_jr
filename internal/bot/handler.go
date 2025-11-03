package bot

import (
	"log"
	"strings"

	"bronivik/internal/config"
	"bronivik/internal/database"
	"bronivik/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot        *tgbotapi.BotAPI
	config     *config.Config
	items      []models.Item
	db         *database.DB
	userStates map[int64]*models.UserState
}

func NewBot(token string, config *config.Config, items []models.Item, db *database.DB) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot:        botAPI,
		config:     config,
		items:      items,
		db:         db,
		userStates: make(map[int64]*models.UserState),
	}, nil
}

const (
	StateMainMenu     = "main_menu"
	StateSelectItem   = "select_item"
	StateSelectDate   = "select_date"
	StateViewSchedule = "view_schedule"
	StatePersonalData = "personal_data"
	StatePhoneNumber  = "phone_number"
	StateConfirmation = "confirmation"
)

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	for update := range updates {
		if update.CallbackQuery != nil {
			b.handleCallbackQuery(update)
			continue
		}

		if update.Message == nil {
			continue
		}

		// Проверка черного списка
		if b.isBlacklisted(update.Message.From.ID) {
			continue
		}

		b.handleMessage(update)
	}
}

func (b *Bot) handleMessage(update tgbotapi.Update) {
	userID := update.Message.From.ID
	text := update.Message.Text

	// Проверка на менеджера
	if b.isManager(userID) {
		if strings.HasPrefix(text, "/export") {
			b.handleExport(update)
			return
		}
	}

	state := b.getUserState(userID)

	switch {
	case text == "/start":
		b.handleMainMenu(update)

	case text == "💼 Доступные позиции":
		b.showAvailableItems(update)

	case text == "📅 Посмотреть расписание":
		b.handleViewSchedule(update)

	case text == "📋 Создать заявку":
		b.handleSelectItem(update)

	case text == "📅 7 дней":
		b.showWeekSchedule(update)

	case text == "🗓 Выбрать дату":
		b.requestSpecificDate(update)

	case text == "⬅️ Назад":
		b.handleMainMenu(update)

	case update.Message.Contact != nil:
		b.handleContactReceived(update)

	case state != nil && state.CurrentStep == StateSelectItem && strings.HasPrefix(text, "🏢 "):
		itemName := strings.TrimPrefix(text, "🏢 ")
		b.handleItemSelection(update, itemName)

	case state != nil && state.CurrentStep == StatePersonalData && text == "✅ Даю согласие":
		b.handlePhoneRequest(update)

	case state != nil && state.CurrentStep == StatePhoneNumber:
		if update.Message.Contact != nil {
			b.handlePhoneReceived(update, update.Message.Contact.PhoneNumber)
		} else {
			// Позволяем ввести номер вручную
			b.handlePhoneReceived(update, text)
		}

	case state != nil && state.CurrentStep == StateConfirmation && text == "✅ Подтвердить заявку":
		b.finalizeBooking(update)

	case text == "❌ Отмена":
		b.handleMainMenu(update)

	default:
		// Обработка дат и других вводов
		if state != nil {
			b.handleCustomInput(update, state)
		} else {
			b.handleMainMenu(update)
		}
	}
}
