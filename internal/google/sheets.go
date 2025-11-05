package google

//
// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"os"
//
// 	"golang.org/x/oauth2/google"
// 	"google.golang.org/api/option"
// 	"google.golang.org/api/sheets/v4"
// )
//
// type SheetsService struct {
// 	service *sheets.Service
// }
//
// func NewSheetsService(credentialsFile string) (*SheetsService, error) {
// 	ctx := context.Background()
//
// 	// Читаем файл учетных данных
// 	b, err := os.ReadFile(credentialsFile)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to read credentials file: %v", err)
// 	}
//
// 	// Настраиваем JWT конфиг
// 	config, err := google.JWTConfigFromJSON(b, sheets.SpreadsheetsScope)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to parse credentials: %v", err)
// 	}
//
// 	// Создаем клиент и сервис
// 	client := config.Client(ctx)
// 	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to create Sheets service: %v", err)
// 	}
//
// 	return &SheetsService{service: srv}, nil
// }
//
// // UpdateUsersSheet обновляет таблицу пользователей
// func (s *SheetsService) UpdateUsersSheet(spreadsheetID string, users []*User) error {
// 	// Подготавливаем данные
// 	var values [][]interface{}
//
// 	// Заголовки
// 	headers := []interface{}{"ID", "Telegram ID", "Username", "First Name", "Last Name", "Phone", "Is Manager", "Is Blacklisted", "Language Code", "Last Activity", "Created At"}
// 	values = append(values, headers)
//
// 	// Данные пользователей
// 	for _, user := range users {
// 		row := []interface{}{
// 			user.ID,
// 			user.TelegramID,
// 			user.Username,
// 			user.FirstName,
// 			user.LastName,
// 			user.Phone,
// 			user.IsManager,
// 			user.IsBlacklisted,
// 			user.LanguageCode,
// 			user.LastActivity.Format("2006-01-02 15:04:05"),
// 			user.CreatedAt.Format("2006-01-02 15:04:05"),
// 		}
// 		values = append(values, row)
// 	}
//
// 	// Обновляем лист "Users"
// 	rangeData := "Users!A1"
// 	valueRange := &sheets.ValueRange{
// 		Values: values,
// 	}
//
// 	_, err := s.service.Spreadsheets.Values.Update(spreadsheetID, rangeData, valueRange).
// 		ValueInputOption("RAW").
// 		Do()
//
// 	return err
// }
//
// // UpdateBookingsSheet обновляет таблицу бронирований
// func (s *SheetsService) UpdateBookingsSheet(spreadsheetID string, bookings []*Booking) error {
// 	var values [][]interface{}
//
// 	// Заголовки для бронирований
// 	headers := []interface{}{"ID", "User ID", "Item ID", "Date", "Status", "User Name", "User Phone", "Item Name", "Created At", "Updated At"}
// 	values = append(values, headers)
//
// 	// Данные бронирований
// 	for _, booking := range bookings {
// 		row := []interface{}{
// 			booking.ID,
// 			booking.UserID,
// 			booking.ItemID,
// 			booking.Date.Format("2006-01-02"),
// 			booking.Status,
// 			booking.UserName,
// 			booking.UserPhone,
// 			booking.ItemName,
// 			booking.CreatedAt.Format("2006-01-02 15:04:05"),
// 			booking.UpdatedAt.Format("2006-01-02 15:04:05"),
// 		}
// 		values = append(values, row)
// 	}
//
// 	// Обновляем лист "Bookings"
// 	rangeData := "Bookings!A1"
// 	valueRange := &sheets.ValueRange{
// 		Values: values,
// 	}
//
// 	_, err := s.service.Spreadsheets.Values.Update(spreadsheetID, rangeData, valueRange).
// 		ValueInputOption("RAW").
// 		Do()
//
// 	return err
// }
//
// // AppendBooking добавляет новое бронирование в конец таблицы
// func (s *SheetsService) AppendBooking(spreadsheetID string, booking *Booking) error {
// 	row := []interface{}{
// 		booking.ID,
// 		booking.UserID,
// 		booking.ItemID,
// 		booking.Date.Format("2006-01-02"),
// 		booking.Status,
// 		booking.UserName,
// 		booking.UserPhone,
// 		booking.ItemName,
// 		booking.CreatedAt.Format("2006-01-02 15:04:05"),
// 		booking.UpdatedAt.Format("2006-01-02 15:04:05"),
// 	}
//
// 	rangeData := "Bookings!A:A" // Автоматически определит последнюю строку
// 	valueRange := &sheets.ValueRange{
// 		Values: [][]interface{}{row},
// 	}
//
// 	_, err := s.service.Spreadsheets.Values.Append(spreadsheetID, rangeData, valueRange).
// 		ValueInputOption("RAW").
// 		InsertDataOption("INSERT_ROWS").
// 		Do()
//
// 	return err
// }
