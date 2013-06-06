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
	gold uint32
	conn *websocket.Conn
	// each outgoing message
	msg_output_queue chan string
	man *Manager
}

func (player *Player) ping() {
	player.send_msg ("ping")
}

func (player *Player) send_msg(msg_to_send string) {
	// put a message in the channel to send
	player.msg_output_queue <- msg_to_send
}


const PLAYER_POSITION_MESSAGE_TYPE = "player_position_message"
const PLAYER_LOGOUT_MESSAGE_TYPE = "player_disconnected_message"
type PlayerPositionMessage struct {
	MsgType string
	ID uint32
	X,Y,Z float64
}

type PlayerLogoutMessage struct {
	MsgType string
	ID uint32
}


/** Gold messages *to* player */
const TO_PLAYER_GOLD_MESSAGE_TYPE = "to_player_gold_message"	
type AddedGoldSubmessage struct {
	StashId GoldStashId
	Amt uint32
	X,Y,Z float64
}
type DeletedGoldSubmessage struct {
	StashId GoldStashId
}
type ChangedGoldSubmessage struct {
	StashId GoldStashId
	FinalAmt uint32
}
type GoldMessage struct {
	MsgType string
	AddedSubmessages [] AddedGoldSubmessage
	DeletedSubmessages [] DeletedGoldSubmessage
	ChangedSubmessages [] ChangedGoldSubmessage
}


/*** Gold messages *from* player */
const PLAYER_GOLD_MESSAGE_TYPE = "gold_message"
const PLAYER_GOLD_MESSAGE_TYPE_GRAB = "grab_gold"
const PLAYER_GOLD_MESSAGE_TYPE_DROP = "drop_gold"
const PLAYER_GOLD_MESSAGE_TYPE_DEDUCT = "deduct_gold"
const PLAYER_GOLD_MESSAGE_TYPE_ADD = "add_gold"
type PlayerGoldMessage struct {
	// Messages can be from dumping gold on ground, trying to pick
	// gold up off ground, buying something with gold, or just
	// adding gold to the world.
	MsgType string
	GoldMsgType string
	ID uint32
	Amt uint32
	X, Y, Z float64
}


func (player *Player) try_decode_player_gold_msg(msg string) bool{
	var pgm PlayerGoldMessage
	err := json.Unmarshal([]byte(msg),&pgm)
	if err != nil {
		log.Println(err)
		return false
	}

	if pgm.MsgType != PLAYER_GOLD_MESSAGE_TYPE {
		return false
	}
	pgm.ID = player.id
	player.man.receive_player_gold_message(pgm)

	log.Println("Received gold message")
	return true
}



func (player *Player) try_decode_player_position_msg(msg string) bool{
	var ppm PlayerPositionMessage
	err := json.Unmarshal([]byte(msg),&ppm)
	if err != nil {
		log.Println(err)
		return false
	}

	if ppm.MsgType != PLAYER_POSITION_MESSAGE_TYPE {
		return false
	}
	ppm.ID = player.id
	player.pos.x = ppm.X
	player.pos.y = ppm.Y
	player.pos.z = ppm.Z	
	player.man.receive_position_update(ppm)
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
		
		if player.try_decode_player_position_msg(message) {
		} else if player.try_decode_player_gold_msg(message) {
		} else{
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
