package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/morodik/chat-monitor/internal/drivers"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка при загрузке .env файла")
	}

	// Читаем токен
	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		log.Fatal("TOKEN не найден в переменных окружения")
	}
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println("Error: ", err)
	}

	ctx := context.Background()
	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		panic(err)
	}

	for update := range updates {
		if update.Message != nil {
			msg := update.Message
			fmt.Println("Сообщение от ", msg.From.Username, msg.Text)
			switch msg.Text {
			case "/start", "/stream":
				keyboard := telegoutil.InlineKeyboard(
					telegoutil.InlineKeyboardRow(
						telegoutil.InlineKeyboardButton("Twitch").WithCallbackData("twitch"),
					),
					telegoutil.InlineKeyboardRow(
						telegoutil.InlineKeyboardButton("Другое").WithCallbackData("other"),
					),
				)

				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID:      telego.ChatID{ID: msg.Chat.ID},
					Text:        "Выберите платформу:",
					ReplyMarkup: keyboard,
				})
				continue
			}
			if strings.HasPrefix(msg.Text, "http") {
				err := CheckUrl(msg.Text)
				if err != nil {
					bot.SendMessage(ctx, &telego.SendMessageParams{
						ChatID: telego.ChatID{ID: msg.Chat.ID},
						Text:   "Неверная ссылка",
					})
				}
			} else {
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: msg.Chat.ID},
					Text:   "Это не ссылка",
				})
			}
			if update.CallbackQuery != nil {
				cb := update.CallbackQuery
				switch cb.Data {
				case "twitch":
					bot.SendMessage(ctx, &telego.SendMessageParams{
						ChatID: telego.ChatID{ID: msg.Chat.ID},
						Text:   "Введите ник:",
					})
					drivers.PlatformCheck("twitch")
					continue
				case "other":
					drivers.PlatformCheck("other")
				}
			}
		}

	}
}

func CheckUrl(streamURL string) error {
	_, err := url.ParseRequestURI(streamURL)
	if err != nil {
		return fmt.Errorf("Невалидный url", err)
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Head(streamURL)
	if err != nil {
		return fmt.Errorf("не удалось подключиться: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	return nil
}
