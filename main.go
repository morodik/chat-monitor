package main

func main() {
	//проверка на использование вебсокета

	//drivers.CheckWebsocket("wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false")

	// это подключение к любому чату не используя веб сокеты и тп

	//drivers.ParseChat("https://rutube.ru/video/0f4b436587fe053673a3a213874e4743/")

	//тут рабочее и стабильно подключение к чату твича

	// driver := drivers.NewTwitchDriver("des0ut")
	// err := driver.Connect()
	// if err != nil {
	// 	panic(err)
	// }

	// msgChan := make(chan string)

	// go func() {
	// 	err := driver.ListenMessage(msgChan)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()
	// for msg := range msgChan {
	// 	fmt.Println("Сообщение:", msg)
	// }
}
