package drivers

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type TwitchDriver struct {
	channel string
	conn    net.Conn
}

func NewTwitchDriver(channel string) *TwitchDriver {
	return &TwitchDriver{channel: channel}
}

func (d *TwitchDriver) Connect() error {
	var err error
	d.conn, err = net.Dial("tcp", "irc.chat.twitch.tv:6667")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(d.conn, "PASS oauth:anonymous\r\n")
	_, _ = fmt.Fprintf(d.conn, "NICK justinfan123\r\n")
	_, _ = fmt.Fprintf(d.conn, "JOIN #%s\r\n", d.channel)

	return nil
}

func (d *TwitchDriver) ListenMessage(out chan string) error {
	reader := bufio.NewReader(d.conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		if strings.Contains(line, "PRIVMSG") {
			parts := strings.Split(line, "PRIVMSG")
			if len(parts) > 1 {
				msg := parts[1]
				msg = strings.SplitN(msg, ":", 2)[1]
				out <- strings.TrimSpace(msg)
			}
		}
	}
}

func (d *TwitchDriver) Close() error {
	return d.conn.Close()
}
