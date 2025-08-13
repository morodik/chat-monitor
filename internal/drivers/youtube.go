package drivers

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Message struct {
	Author    string
	Content   string
	Timestamp string
}

type YouTubeDriver struct {
	videoID       string
	apiKey        string
	service       *youtube.Service
	chatID        string
	nextPageToken string
	stopChan      chan struct{}
}

func NewYouTubeDriver(url string) *YouTubeDriver {
	apiKey := os.Getenv("APIKEY")
	if apiKey == "" {
		log.Println("APIKEY не найден, используйте .env")
	}
	videoID, err := extractVideoID(url)
	if err != nil {
		log.Printf("Ошибка извлечения videoID: %v", err)
		videoID = "" // или обработать ошибку иначе
	}
	return &YouTubeDriver{
		videoID:  videoID,
		apiKey:   apiKey,
		stopChan: make(chan struct{}),
	}
}

func (d *YouTubeDriver) Connect() error {
	if d.apiKey == "" {
		return fmt.Errorf("API key не указан")
	}
	if d.videoID == "" {
		return fmt.Errorf("videoID не указан или неверная ссылка")
	}

	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(d.apiKey))
	if err != nil {
		return fmt.Errorf("ошибка создания YouTube сервиса: %v", err)
	}
	d.service = service

	chatID, err := d.getLiveChatID()
	if err != nil {
		return err
	}
	d.chatID = chatID
	d.nextPageToken = ""

	return nil
}

func (d *YouTubeDriver) getLiveChatID() (string, error) {
	call := d.service.Videos.List([]string{"liveStreamingDetails"}).Id(d.videoID)
	resp, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("ошибка запроса videos.list: %v", err)
	}
	if len(resp.Items) == 0 || resp.Items[0].LiveStreamingDetails == nil || resp.Items[0].LiveStreamingDetails.ActiveLiveChatId == "" {
		return "", fmt.Errorf("это не активная live-трансляция или chatID не найден")
	}
	return resp.Items[0].LiveStreamingDetails.ActiveLiveChatId, nil
}

func (d *YouTubeDriver) ListenMessage(out chan string, stopChan chan struct{}) error {
	if stopChan == nil {
		stopChan = d.stopChan
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stopChan
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			close(out)
			return nil
		default:
			call := d.service.LiveChatMessages.List(d.chatID, []string{"snippet", "authorDetails"}).PageToken(d.nextPageToken)
			reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
			resp, err := call.Context(reqCtx).Do()
			reqCancel()
			if err != nil {
				log.Printf("Ошибка polling: %v. Ждём 5 сек и retry...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, item := range resp.Items {
				msg := fmt.Sprintf("[%s] %s: %s", item.Snippet.PublishedAt, item.AuthorDetails.DisplayName, item.Snippet.DisplayMessage)
				out <- msg
			}

			d.nextPageToken = resp.NextPageToken
			sleepDuration := time.Duration(resp.PollingIntervalMillis) * time.Millisecond
			if sleepDuration < 5*time.Second {
				sleepDuration = 5 * time.Second
			}
			time.Sleep(sleepDuration)
		}
	}
}

func (d *YouTubeDriver) Close() {
	close(d.stopChan)
}

func extractVideoID(url string) (string, error) {
	re := regexp.MustCompile(`v=([a-zA-Z0-9_-]{11})`)
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", fmt.Errorf("не удалось извлечь ID видео из ссылки")
	}
	return match[1], nil
}
