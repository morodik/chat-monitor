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
	Driver   drivers.Driver
	StopChan chan struct{}
	MsgChan  chan string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка при загрузке .env файла")
	}

	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		log.Fatal("TOKEN не найден в переменных окружения")
	}

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	ctx := context.Background()
	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Ошибка запуска long polling: %v", err)
	}

	userState := make(map[int64]*Session)

	for update := range updates {
		// бработка текстовых сообщений
		if update.Message != nil {
			msg := update.Message
			chatID := msg.Chat.ID
			fmt.Printf("Сообщение от @%s: %s\n", msg.From.Username, msg.Text)

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
					ChatID:      telego.ChatID{ID: chatID},
					Text:        "Выберите платформу:",
					ReplyMarkup: keyboard,
				})
				continue
			}

			session, ok := userState[chatID]
			if ok && session != nil {
				switch session.State {
				case "awaiting_twitch_nik":
					//завершаем предыдущую сессию, если есть

					driver := drivers.NewTwitchDriver(msg.Text)
					err := driver.Connect()
					if err != nil {
						log.Printf("Ошибка подключения к Twitch: %v", err)
						bot.SendMessage(ctx, &telego.SendMessageParams{
							ChatID: telego.ChatID{ID: chatID},
							Text:   "Не удалось подключиться к Twitch.",
						})
						continue
					}
					keyboard := telegoutil.InlineKeyboard(telegoutil.InlineKeyboardRow(
						telegoutil.InlineKeyboardButton("Завершить").WithCallbackData("end"),
					))
					bot.SendMessage(ctx, &telego.SendMessageParams{
						ChatID:      telego.ChatID{ID: chatID},
						Text:        "Успешное подключение, при проблемах с трансляцией вы получите уведомление.",
						ReplyMarkup: keyboard,
					})

					stopChan := make(chan struct{})
					msgChan := make(chan string)

					//сохраняем новую сессию
					userState[chatID] = &Session{
						State:    "", // сбрасываем состояние
						Driver:   driver,
						StopChan: stopChan,
						MsgChan:  msgChan,
					}

					//запуск прослушивания чата
					go driver.ListenMessage(msgChan, stopChan)
					go func() {
						for twitchMsg := range msgChan {
							fmt.Printf("[Twitch %s] %s\n", msg.Text, twitchMsg)
						}
					}()

				case "awaiting_link":
					driver := drivers.NewOtherDriver(msg.Text)
					err := driver.Connect()
					if err != nil {
						log.Printf("Ошибка подключения к трансляции: %v", err)
						bot.SendMessage(ctx, &telego.SendMessageParams{
							ChatID: telego.ChatID{ID: chatID},
							Text:   "Не удалось подключиться к трансляции.",
						})
						continue
					}
					if strings.HasPrefix(msg.Text, "http") {
						err := CheckUrl(msg.Text)
						if err != nil {
							bot.SendMessage(ctx, &telego.SendMessageParams{
								ChatID: telego.ChatID{ID: chatID},
								Text:   "Неверная ссылка.",
							})
						} else {
							bot.SendMessage(ctx, &telego.SendMessageParams{
								ChatID: telego.ChatID{ID: chatID},
								Text:   "Ссылка успешно принята.",
							})
						}
					} else {
						bot.SendMessage(ctx, &telego.SendMessageParams{
							ChatID: telego.ChatID{ID: chatID},
							Text:   "Это не ссылка.",
						})
					}
					stopChan := make(chan struct{})
					msgChan := make(chan string)
					userState[chatID] = &Session{
						State:    "",
						Driver:   driver,
						StopChan: stopChan,
						MsgChan:  msgChan,
					}

					go driver.ListenMessage(msgChan, stopChan)

					go func() {
						for msg := range msgChan {
							fmt.Println("[Other]", msg)
						}
					}()
				}
			}
		}

		// обработка нажатия на кнопку
		if update.CallbackQuery != nil {
			cb := update.CallbackQuery
			chatID := cb.Message.GetChat().ID

			// Завершаем текущую сессию, если есть
			if session, ok := userState[chatID]; ok {
				stopSession(session)
			}

			switch cb.Data {
			case "twitch":
				userState[chatID] = &Session{
					State:    "awaiting_twitch_nik",
					Driver:   nil,
					StopChan: nil,
					MsgChan:  nil,
				}
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "Введите ник стримера:",
				})
			case "other":
				userState[chatID] = &Session{
					State:    "awaiting_link",
					Driver:   nil,
					StopChan: nil,
					MsgChan:  nil,
				}
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "Отправьте ссылку на стрим:",
				})
			case "end":
				if session, ok := userState[chatID]; ok {
					if session.StopChan != nil {
						close(session.StopChan)
					}
					if session.MsgChan != nil {
						close(session.MsgChan)
					}
					if session.Driver != nil {
						session.Driver.Close()
					}
				}
				userState[chatID] = &Session{
					State:    "",
					Driver:   nil,
					StopChan: nil,
					MsgChan:  nil,
				}
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "Сессия завершена",
				})
				continue
			}
		}
	}
}

func CheckUrl(streamURL string) error {
	_, err := url.ParseRequestURI(streamURL)
	if err != nil {
		return fmt.Errorf("невалидный URL")
	}

	client := http.Client{Timeout: 5 * time.Second}
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

func stopSession(session *Session) {
	if session == nil {
		return
	}
	if session.StopChan != nil {
		close(session.StopChan)
		session.StopChan = nil
	}
	if session.MsgChan != nil {
		close(session.MsgChan)
		session.MsgChan = nil
	}
	if session.Driver != nil {
		session.Driver.Close()
		session.Driver = nil
	}
}
