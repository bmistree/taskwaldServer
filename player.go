package main

import "log"
import "code.google.com/p/go.net/websocket"
import "encoding/json"


type Position struct {
	x, y, z float64
}

type Player struct {
	id uint32
	pos Position
	conn *websocket.Conn
	// each outgoing message
	msg_output_queue chan string
}

func (player *Player) ping() {
	// put a message in the channel to send
	player.msg_output_queue <- "ping"
}


type PlayerPositionMessage struct {
	X,Y,Z float64 
}

func try_decode_player_position_msg(msg string) bool{
	var ppm PlayerPositionMessage
	err := json.Unmarshal([]byte(msg),&ppm)
	if err != nil {
		log.Println(err)
		return false
	}

	log.Println("Received position message")
	return true
}


// should be three types of messages:
// 1) this is my position
// 2) attacking
// 3) going into world...
func (player * Player) read_msg_loop() {
	// Infinite loop while reading
	for {
		var message string
		err := websocket.Message.Receive(player.conn, &message)
		if err != nil {
			// likely that connection went down.  cleanup
			// handled by defer of web socket connection handler.				
			break
		}
		
		if try_decode_player_position_msg(message) {
		} else {
			log.Println("Unknown message type")
		}		
	}
	player.conn.Close()
}

func (player *Player) write_msg_loop() {
	for msg_to_send := range player.msg_output_queue {
		err := websocket.Message.Send(player.conn,msg_to_send)
		if err != nil {
			// likely that connection went down.  cleanup
			// handled by defer of web socket connection handler.
			break
		}
	}
	// done sending messages.  close connection
	player.conn.Close()
}
