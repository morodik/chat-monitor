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

type OtherDriver struct {
	url      string
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}
}

func NewOtherDriver(url string) *OtherDriver {
	return &OtherDriver{
		url:      url,
		stopChan: make(chan struct{}),
	}
}

func (d *OtherDriver) Connect() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) "+
			"Chrome/125.0.0.0 Safari/537.36"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	d.ctx = ctx
	d.cancel = func() {
		cancel()
		cancelAlloc()
	}

	// Навигация и дальнейшая логика:
	if err := chromedp.Run(d.ctx, chromedp.Navigate(d.url)); err != nil {
		return fmt.Errorf("ошибка навигации: %w", err)
	}

	return nil
}
func (d *OtherDriver) ListenMessage(out chan string, stopChan chan struct{}) error {
	defer close(out)

	var chatTexts []string

	// Регулярные выражения для фильтрации нежелательных сообщений
	timeRegex := regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	timestampRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	numberRegex := regexp.MustCompile(`^[\d,]+$`)
	systemMessageRegex := regexp.MustCompile(`(?i)(авторизуйтесь|добро пожаловать|настройки чата|to pick up|while dragging|press space)`)

	for {
		select {
		case <-d.stopChan:
			d.cancel()
			log.Println("Парсинг OtherDriver остановлен.")
			return nil
		default:
			var lastSnapshot string
			err := chromedp.Run(d.ctx,
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

				// Фильтрация
				if line == "" || strings.HasSuffix(line, ":") {
					continue
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

				out <- line
				chatTexts = append(chatTexts, line)
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (d *OtherDriver) Close() {
	close(d.stopChan)
	d.cancel()

}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
