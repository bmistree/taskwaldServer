package player

import "log"

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
		log.Println("Received message: " + message)
		// h.broadcast <- message
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
