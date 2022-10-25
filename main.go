package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"
	//"encoding/json"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "zkillboard.com", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/websocket/"}
	log.Printf("connecting to %s", u.String())

	c, dialResp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dialResp:", dialResp)
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()
	
	subMsg := []byte("{\"action\":\"sub\",\"channel\":\"region:10000060\"}")
	log.Println("Subscribing")
	err = c.WriteMessage(1, subMsg)
	if err != nil {
		log.Println("SubError:", err)
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}