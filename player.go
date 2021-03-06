package main

import "log"
import "code.google.com/p/go.net/websocket"
import "encoding/json"
import "math"

const POINT_EXPONENT float64 = .0092
const HIT_COST float64 = .5
const HIT_GAIN float64 = .25

type Position struct {
	x, y, z float64
}

func (pos * Position) make_copy () Position{
	var pos_copy Position
	pos_copy.x = pos.x
	pos_copy.y = pos.y
	pos_copy.z = pos.z
	return pos_copy
}

type Player struct {
	id uint32
	pos Position
	gold uint32
	points float64
	conn *websocket.Conn
	// each outgoing message
	msg_output_queue chan string
	man *Manager
	plant_manager *PlantManagerSingleton
}

func (player *Player) ping() {
	player.send_msg ("ping")
}

func (player *Player) send_msg(msg_to_send string) {
	// put a message in the channel to send
	player.msg_output_queue <- msg_to_send
}

func abs(val int32) int32{
	if val < 0 {
		return -val
	}
	return val
}

func (player * Player) get_hit () {
	player.points -= HIT_COST
	player.send_player_data_update()
}

func (player * Player) hit_someone_else () {
	player.points += HIT_GAIN
	player.send_player_data_update()
}

func (player * Player) send_player_data_update () {
	var pdm PlayerDataMessage
	pdm.MsgType = PLAYER_DATA_MESSAGE_TYPE
	pdm.GoldAmt = player.gold
	pdm.Points = player.points
	pdm.ID = player.id
	pdm.Me = true
	
	byter, _ := json.Marshal(pdm)
	msg_string := string(byter)
	player.msg_output_queue <- msg_string

	pdm.Me = false
	byter, _ = json.Marshal(pdm)
	msg_string = string(byter)

	// notify everyone that this player's score has
	// changed so that they can update their scoreboards.
	player.man.add_msg_for_broadcast(msg_string)
}



// Changes gold by amt_to_change_by (note, this value can be
// negative), while assuring always have >=0 gold on player.  Returns
// change amount.
// change_score is true if we are spending the gold on our
// score... ie, we should update our score as well.
func (player *Player) change_gold (amt_to_change_by int32, change_score bool) uint32 {

	// Actually determine how much gold the player will have left
	
	var player_gold int32
	player_gold = int32(player.gold)
	
	var delta uint32
	delta = 0
	
	if player_gold + amt_to_change_by >= 0 {
		delta = uint32(abs(amt_to_change_by))
		
		player_gold += amt_to_change_by
		player.gold = uint32(abs(player_gold))
	} else {
		delta = uint32(player_gold)
		player.gold = 0
	}

	// update score if spent gold on points
	// FIXME: refactor into own function so it's more pluggable.
	if change_score {
		float_delta := float64(delta)
		// should guarantee that spending 500 gold gives 100 points
		player.points += math.Exp(POINT_EXPONENT*float_delta)
	}

	
	// send final gold value to player for how much gold will have
	// left.

	var pdm PlayerDataMessage
	pdm.MsgType = PLAYER_DATA_MESSAGE_TYPE
	pdm.GoldAmt = player.gold
	pdm.Points = player.points
	pdm.ID = player.id
	pdm.Me = true
	
	byter, _ := json.Marshal(pdm)
	msg_string := string(byter)
	player.msg_output_queue <- msg_string

	pdm.Me = false
	byter, _ = json.Marshal(pdm)
	msg_string = string(byter)

	if change_score {
		// notify everyone that this player's score has
		// changed so that they can update their scoreboards.
		player.man.add_msg_for_broadcast(msg_string)
	}
	
	return delta
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

/** Both to and from player */
const PLAYER_PLANT_MESSAGE_TYPE = "player_plant_message"
type PlayerPlantMessage struct {
	MsgType string
	X,Y,Z float64
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

const FIRE_MESSAGE_TYPE = "fire_message"
type FireMessage struct {
	MsgType string
        Dest_x, Dest_y, Dest_z float64
	OpponentHitID uint32
	ShooterID uint32
	player *Player
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

/*** Messages about the amount of gold the player has */
const PLAYER_DATA_MESSAGE_TYPE = "player_data_message"
type PlayerDataMessage struct {
	MsgType string
	GoldAmt uint32
	ID uint32
	Points float64
	// true if the message is a change to the player that
	// initiated the event that started the message.  false
	// otherwise.
	Me bool
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
	return true
}


func (player * Player) try_decode_player_plant_msg (msg string) bool {
	var player_plant_message PlayerPlantMessage
	err := json.Unmarshal([]byte(msg),&player_plant_message)
	if err != nil {
		log.Println(err)
		return false
	}

	if player_plant_message.MsgType != PLAYER_PLANT_MESSAGE_TYPE {
		return false
	}
	player.plant_manager.add_plant(player_plant_message)
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

func (player * Player) try_decode_fire_message (msg string) bool {
	var fm FireMessage
	err := json.Unmarshal([]byte(msg),&fm)
	if err != nil {
		log.Println(err)
		return false
	}
	
	if fm.MsgType != FIRE_MESSAGE_TYPE {
		return false
	}
	
	fm.player = player
	player.man.receive_fire_message(fm)
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
		} else if player.try_decode_player_plant_msg(message) {
		} else if player.try_decode_fire_message (message) {
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

func (player * Player) buy_points (amt_to_spend uint32,gold_manager * GoldManagerSingleton) {
	if amt_to_spend > player.gold {
		// need to try to grab rest of gold from environment... if can
		diff := amt_to_spend - player.gold
		amt_grabbed := gold_manager.grab_gold(player,diff,player.pos)
		amt_to_spend = amt_grabbed + player.gold
	}
	
	var int_amt_to_spend int32
	int_amt_to_spend = int32(amt_to_spend)
	player.change_gold(-int_amt_to_spend,true)
}
