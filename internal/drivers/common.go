package drivers

type Driver interface {
	Connect() error
	ListenMessage(out chan string, stopChan chan struct{}) error
	Close()
}
