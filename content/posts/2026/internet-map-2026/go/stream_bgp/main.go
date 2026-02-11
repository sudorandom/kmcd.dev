package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// RIPE RIS Live WebSocket URL
const risLiveURL = "wss://ris-live.ripe.net/v1/ws/?client=kmcd-internet-map"

// Message defines the structure of the JSON messages we receive
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	// Handle Ctrl+C gracefully
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Printf("Connecting to %s...\n", risLiveURL)

	c, _, err := websocket.DefaultDialer.Dial(risLiveURL, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Subscribe to the firehose (all messages)
	// You can filter this! e.g., {"host": "rrc21"} for a specific collector
	subscribeMsg := map[string]interface{}{
		"type": "ris_subscribe",
		"data": map[string]interface{}{
			"moreSpecific": true,
			"type":         "UPDATE", // Only show route updates
		},
	}

	if err := c.WriteJSON(subscribeMsg); err != nil {
		log.Fatal("subscribe:", err)
	}

	fmt.Println("Connected! Streaming global BGP updates...")
	fmt.Println("------------------------------------------------")

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			var msg Message
			err := c.ReadJSON(&msg)
			if err != nil {
				log.Println("read:", err)
				return
			}

			// We only care about BGP UPDATE messages 
			if msg.Type == "ris_message" {
				path := msg.Data["path"]
				prefix := msg.Data["announcements"]
				
				// Handle withdrawals (routes being removed)
				if prefix == nil {
					prefix = "WITHDRAWAL"
				}

				// Print the timestamp, the route prefix, and the AS path
				fmt.Printf("[%s] Prefix: %v | Path: %v\n", 
					time.Now().Format("15:04:05"), 
					prefix, 
					path,
				)
			}
		}
	}()

	// Wait for interrupt
	<-interrupt
	fmt.Println("\nDisconnecting...")
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}
