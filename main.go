package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// TODO: Either have a function call inside goroutine or use a message channel , or goroutine inside a goroutine?
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

func postZKill(s *discordgo.Session, chanID string, kill zKBlock) {
	/*
		GroupID's
		Titan: 30
		Supercarrier: 659
		Carrier: 547
		Dreadnought: 485
		FAX: 1538

	*/

	var discordMessage discordgo.MessageSend
	switch kill.ShipGroupID {
	case 30, 659, 547, 485, 1538:
		discordMessage.Content = kill.URL
		s.ChannelMessageSendComplex(chanID, &discordMessage)
	}
	discordMessage.Content = kill.URL
	s.ChannelMessageSendComplex(chanID, &discordMessage)

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	discordToken := os.Getenv("BOT_TOKEN")
	discordChannel := os.Getenv("PERSONAL_CHAN")
	fmt.Printf("ChannelID: %T\n", discordChannel)
	log.Println("ID", discordChannel)
	//discordChannel = strconv.Itoa(discordChannel)
	//guildToken := os.Getenv("GUILD_TOKEN")
	//appToken := os.Getenv("APP_TOKEN")

	flag.Parse()
	log.SetFlags(0)

	s, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal("Invalid bot:", err)
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages
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

	var zResponse zKBlock

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)

			json.Unmarshal(message, &zResponse)
			if zResponse.KillID != 0 {
				go postZKill(s, discordChannel, zResponse)
			}
			zResponse.KillID = 0

		}
	}()
	_, err = s.ChannelMessageSend(discordChannel, "Listening")
	if err != nil {
		log.Println("DiscordSend:", err)
	}

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
