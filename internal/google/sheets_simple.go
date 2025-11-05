package google

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"bronivik/internal/models"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsService struct {
	service         *sheets.Service
	usersSheetID    string
	bookingsSheetID string
}

func NewSimpleSheetsService(credentialsFile, usersSheetID, bookingsSheetID string) (*SheetsService, error) {
	ctx := context.Background()

	// Читаем файл учетных данных сервисного аккаунта
	credentialsJSON, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %v", err)
	}

	// Создаем JWT конфигурацию
	config, err := google.JWTConfigFromJSON(credentialsJSON, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %v", err)
	}

	// Создаем клиент
	client := config.Client(ctx)

	// Создаем сервис
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Sheets service: %v", err)
	}

	return &SheetsService{
		service:         srv,
		usersSheetID:    usersSheetID,
		bookingsSheetID: bookingsSheetID,
	}, nil
}

// TestConnection проверяет подключение к таблице
func (s *SheetsService) TestConnection() error {
	// Пробуем прочитать первую ячейку таблицы пользователей
	_, err := s.service.Spreadsheets.Values.Get(s.usersSheetID, "Users!A1").Do()
	if err != nil {
		return fmt.Errorf("connection test failed: %v", err)
	}
	return nil
}

// GetServiceAccountEmail возвращает email сервисного аккаунта
func (s *SheetsService) GetServiceAccountEmail(credentialsFile string) (string, error) {
	file, err := os.ReadFile(credentialsFile)
	if err != nil {
		return "", err
	}

	var creds struct {
		ClientEmail string `json:"client_email"`
	}

	if err := json.Unmarshal(file, &creds); err != nil {
		return "", err
	}

	return creds.ClientEmail, nil
}

// UpdateUsersSheet обновляет таблицу пользователей
func (s *SheetsService) UpdateUsersSheet(users []*models.User) error {
	// Подготавливаем данные
	var values [][]interface{}

	// Заголовки
	headers := []interface{}{"ID", "Telegram ID", "Username", "First Name", "Last Name", "Phone", "Is Manager", "Is Blacklisted", "Language Code", "Last Activity", "Created At"}
	values = append(values, headers)

	// Данные пользователей
	for _, user := range users {
		row := []interface{}{
			user.ID,
			user.TelegramID,
			user.Username,
			user.FirstName,
			user.LastName,
			user.Phone,
			user.IsManager,
			user.IsBlacklisted,
			user.LanguageCode,
			user.LastActivity.Format("2006-01-02 15:04:05"),
			user.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		values = append(values, row)
	}

	// Полностью очищаем и перезаписываем лист
	rangeData := "Users!A1:K" + fmt.Sprintf("%d", len(values))
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	// Используем Overwrite для полной замены данных
	_, err := s.service.Spreadsheets.Values.Update(s.usersSheetID, rangeData, valueRange).
		ValueInputOption("RAW").
		Do()

	return err
}

// AppendBooking добавляет новое бронирование
func (s *SheetsService) AppendBooking(booking *models.Booking) error {
	row := []interface{}{
		booking.ID,
		booking.UserID,
		booking.ItemID,
		booking.Date.Format("2006-01-02"),
		booking.Status,
		booking.UserName,
		booking.Phone,
		booking.ItemName,
		booking.CreatedAt.Format("2006-01-02 15:04:05"),
		booking.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	rangeData := "Bookings!A:A"
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	_, err := s.service.Spreadsheets.Values.Append(s.bookingsSheetID, rangeData, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Do()

	return err
}

// UpdateBookingsSheet обновляет всю таблицу бронирований
func (s *SheetsService) UpdateBookingsSheet(bookings []*models.Booking) error {
	var values [][]interface{}

	// Заголовки
	headers := []interface{}{"ID", "User ID", "Item ID", "Date", "Status", "User Name", "User Phone", "Item Name", "Created At", "Updated At"}
	values = append(values, headers)

	// Данные бронирований
	for _, booking := range bookings {
		row := []interface{}{
			booking.ID,
			booking.UserID,
			booking.ItemID,
			booking.Date.Format("2006-01-02"),
			booking.Status,
			booking.UserName,
			booking.Phone,
			booking.ItemName,
			booking.CreatedAt.Format("2006-01-02 15:04:05"),
			booking.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		values = append(values, row)
	}

	// Полностью очищаем и перезаписываем лист
	rangeData := "Bookings!A1:J" + fmt.Sprintf("%d", len(values))
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := s.service.Spreadsheets.Values.Update(s.bookingsSheetID, rangeData, valueRange).
		ValueInputOption("RAW").
		Do()

	return err
}
