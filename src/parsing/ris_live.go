package parsing

import (
	"BGPAlert/common"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func main() {
	// RIS Live websocket url
	socketUrl := "ws://ris-live.ripe.net/v1/ws/"

	// create websocket connection to ris live websocket
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Websocket connection error:", err)
	}
	defer conn.Close()

	for {
		//keep reading in new message from connection
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Websocket read error:", err)
			return
		}

		bgpMsg, err := parseLiveMessage(message)
		if err != nil {
			log.Println("Error parsing BGP message", err)
		} else {
			fmt.Printf("Parsed BGP Message: %+v\n", bgpMsg) //prints parsed BGP msg
		}
	}
}

func parseLiveMessage(data []byte) (common.BGPMessage, error) {
	var bgpMsg common.BGPMessage
	if err := json.Unmarshal(data, &bgpMsg); err != nil {
		return common.BGPMessage{}, fmt.Errorf("Error parsing JSON: %v", err)
	}
	return bgpMsg, nil
}
