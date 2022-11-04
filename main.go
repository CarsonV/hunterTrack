package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// TODO: make zkill response struct like other.
// check/make bot functionality
var addr = flag.String("addr", "zkillboard.com", "http service address")

type zKillSubMsg struct {
	Action  string `json:"action"`
	Channel string `json:"channel"`
}

type zKBlock struct {
	Action      string `json:"action"`
	KillID      int    `json:"killID"`
	CharID      int    `json:"character_id"`
	CorpID      int    `json:"corporation_id"`
	AllianceID  int    `json:"alliance_id"`
	ShipID      int    `json:"ship_type_id"`
	ShipGroupID int    `json:"group_id"`
	URL         string `json:"url"`
	Hash        string `json:"hash"`
	SubChan     string `json:"channel"`
}

var s *discordgo.Session

func zkillLink(s *discordgo.Session) {

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	discordToken := os.Getenv("BOT_TOKEN")
	//guildToken := os.Getenv("GUILD_TOKEN")
	//appToken := os.Getenv("APP_TOKEN")

	flag.Parse()
	log.SetFlags(0)

	s, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal("Invalid bot:", err)
	}
	//https://pkg.go.dev/github.com/bwmarrin/discordgo#Session.ChannelMessageSendComplex
	err = s.Open()
	if err != nil {
		log.Fatal("Discord: ", err)
	}

	defer s.Close()

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

	/*
		Delve: 10000060
		Querious: 10000050
		Period Basis: 10000063
		Fountain: 10000058
	*/

	regionList := make([]string, 4)
	regionList[0] = "region:10000060"
	regionList[1] = "region:10000050"
	regionList[2] = "region:10000063"
	regionList[3] = "region:10000058"

	for i := range regionList {
		regionSub := zKillSubMsg{
			Action:  "sub",
			Channel: regionList[i],
		}

		newSub, err := json.Marshal(regionSub)
		if err != nil {
			log.Println("marshal: ", err)
		}
		log.Println(string(newSub))
		log.Println("Subscribing: ", i)

		err = c.WriteMessage(1, newSub)
		if err != nil {
			log.Println("SubError:", err)
			return
		}
	}

	//below keeps the websocket goroutine running until clean shutdown or connection close.
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
