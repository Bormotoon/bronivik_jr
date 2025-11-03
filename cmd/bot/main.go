package main

import (
	"log"
	"os"
	"path/filepath"

	"bronivik/internal/bot"
	"bronivik/internal/config"
	"bronivik/internal/database"
	"bronivik/internal/models"
	"gopkg.in/yaml.v2"
)

func main() {
	// Загрузка основной конфигурации
	// configData, err := os.ReadFile("configs/config.yaml")
	// if err != nil {
	// 	log.Fatal("Ошибка чтения config.yaml:", err)
	// }

	// Загрузка конфигурации
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)

	// var cfg config.Config
	// if err := yaml.Unmarshal(configData, &cfg); err != nil {
	// 	log.Fatal("Ошибка парсинга config.yaml:", err)
	// }

	// Загрузка позиций из отдельного файла
	itemsData, err := os.ReadFile("configs/items.yaml")
	if err != nil {
		log.Fatal("Ошибка чтения items.yaml:", err)
	}

	var itemsConfig struct {
		Items []models.Item `yaml:"items"`
	}
	if err := yaml.Unmarshal(itemsData, &itemsConfig); err != nil {
		log.Fatal("Ошибка парсинга items.yaml:", err)
	}

	// Создаем необходимые директории
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		log.Fatal("Ошибка создания директории для базы данных:", err)
	}

	if err := os.MkdirAll(cfg.Exports.Path, 0755); err != nil {
		log.Fatal("Ошибка создания директории для экспорта:", err)
	}

	// Инициализация базы данных
	db, err := database.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatal("Ошибка инициализации базы данных:", err)
	}
	defer db.Close()

	// Устанавливаем items в базу данных
	db.SetItems(itemsConfig.Items)

	if cfg.Telegram.BotToken == "YOUR_BOT_TOKEN_HERE" {
		log.Fatal("Задайте токен бота в config.yaml")
	}

	// Создание и запуск бота
	telegramBot, err := bot.NewBot(cfg.Telegram.BotToken, cfg, itemsConfig.Items, db)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	log.Println("Бот запущен...")
	telegramBot.Start()
}
