package drivers

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка при загрузке .env файла")
	}
	apiKey := os.Getenv("APIKEY")
	if apiKey == "" {
		log.Fatal("APIKEY не найден в переменных окружения")
	}

	var streamURL string
	streamID, err := extractVideoID(streamURL)
	if err != nil {
		log.Fatalf("Ошибка извлечения id", err)
	}
	fmt.Println(streamID)
}

func extractVideoID(url string) (string, error) {
	re := regexp.MustCompile(`v=([a-zA-Z0-9_-]{11})`)
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", fmt.Errorf("не удалось извлечь ID видео из ссылки")
	}
	return match[1], nil
}
