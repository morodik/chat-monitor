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

type Session struct {
	State    string
	Driver   *drivers.TwitchDriver
	StopChan chan struct{}
}

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

	userState := make(map[int64]*Session)

	for update := range updates {
		if update.Message != nil {
			msg := update.Message
			chatID := msg.Chat.ID
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

			state, ok := userState[chatID]
			if ok && state != nil {
				switch state.State {
				case "awaiting_twitch_nik":
					driver := drivers.NewTwitchDriver(msg.Text)
					err := driver.Connect()
					if err != nil {
						log.Println("Ошибка подключения к Twitch:", err)
						bot.SendMessage(ctx, &telego.SendMessageParams{
							ChatID: telego.ChatID{ID: chatID},
							Text:   "Ошибка подключения к Twitch",
						})
						continue
					}
					stopChan := make(chan struct{})
					userState[chatID] = &Session{
						State:    "",
						Driver:   driver,
						StopChan: stopChan,
					}
					msgChan := make(chan string)
					go driver.ListenMessage(msgChan, stopChan)

					go func() {
						for twitchMsg := range msgChan {
							fmt.Println("Сообщение из Twitch:", twitchMsg)
						}
					}()

				case "awaiting_link":
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
				}
			}

		}
		if update.CallbackQuery != nil {
			cb := update.CallbackQuery
			chatID := cb.Message.GetChat().ID
			switch cb.Data {
			case "twitch":
				if session, ok := userState[chatID]; ok && session.StopChan != nil {
					close(session.StopChan)
					session.Driver.Close()
				}
				userState[chatID] = &Session{
					State:    "awaiting_twitch_nik",
					Driver:   nil,
					StopChan: nil,
				}
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: cb.Message.GetChat().ID},
					Text:   "Введите ник:",
				})
				continue
			case "other":
				userState[chatID] = &Session{
					State:    "awaiting_link",
					Driver:   nil,
					StopChan: nil,
				}

				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: cb.Message.GetChat().ID},
					Text:   "Отправьте ссылку:",
				})
				drivers.PlatformCheck("other")
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
