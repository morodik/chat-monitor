package drivers

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func ParseChat(url string) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) "+
			"Chrome/125.0.0.0 Safari/537.36"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Навигация и дальнейшая логика:
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		log.Fatal(err)
	}

	var chatTexts []string
	lastSnapshot := ""

	// Регулярные выражения для фильтрации нежелательных сообщений
	timeRegex := regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	timestampRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	numberRegex := regexp.MustCompile(`^[\d,]+$`)
	systemMessageRegex := regexp.MustCompile(`(?i)(авторизуйтесь|добро пожаловать|настройки чата|to pick up|while dragging|press space)`)

	for {
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`[...document.querySelectorAll("div")]
				.filter(div => div.innerText.trim().length > 0)
				.map(div => div.innerText.trim())
				.join("\n")`, &lastSnapshot),
		)
		if err != nil {
			log.Println("Ошибка при парсинге:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		lines := strings.Split(lastSnapshot, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			if line == "" || strings.HasSuffix(line, ":") {
				continue // ник без сообщения
			}

			if timeRegex.MatchString(line) ||
				timestampRegex.MatchString(line) ||
				numberRegex.MatchString(line) ||
				systemMessageRegex.MatchString(line) {
				continue
			}

			// Убираем ник, если строка в формате "Ник: сообщение"
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				line = strings.TrimSpace(parts[1])
			}

			if len(line) < 3 || contains(chatTexts, line) {
				continue
			}

			fmt.Println("Сообщение:", line)
			chatTexts = append(chatTexts, line)

			// Проверка на дубликаты
			if !contains(chatTexts, line) {
				fmt.Println("Сообщение:", line)
				chatTexts = append(chatTexts, line)
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
