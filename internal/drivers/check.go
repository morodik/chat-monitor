package drivers

import (
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

// func main() {
// 	CheckWebsocket("https://kick.com/pokerok_streams")
// }

func CheckWebsocket(url string) bool {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("Нет веб сокета")
		return false
	} else {
		fmt.Println("есть веб сокет")
	}
	defer conn.Close()
	return true
}

func PlatformCheck(url string) bool {
	url = strings.ToLower(url)
	if strings.HasPrefix(url, "twitch") {
		return true
	}
	return false
}
