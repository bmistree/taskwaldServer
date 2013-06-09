package main

import "log"
import "encoding/json"


/***************** MANAGER DEFINITIONS ********/

type Manager struct {
	// map [<key type>] <value type>
	all_connections map[uint32] *Player
	register_channel chan *Player
	unregister_channel chan *Player
	// FIXME: Should probably use a pointer instead
	position_update_channel chan PlayerPositionMessage

	// messages received by gold stash manager and sent out to all
	// players
	gold_message_channel chan * GoldMessage
	// messages received by players about changes to in-game gold
	// (eg., creating more in-game gold)
	player_gold_message_channel chan PlayerGoldMessage

	// to broadcast to all players
	plant_message_channel chan PlayerPlantMessage

	// probably should have used this for all times when
	// broadcasting a message from the server to player
	broadcast_channel chan string
		
	gold_manager * GoldManagerSingleton
}

func (man * Manager) receive_position_update(ppm PlayerPositionMessage) {
	// only send update messages to those players that are within
	man.position_update_channel <- ppm
}

func (man * Manager) receive_player_gold_message (pgm PlayerGoldMessage) {
	man.player_gold_message_channel <- pgm
}
func (man * Manager) receive_gold_message(gold_message * GoldMessage) {
	man.gold_message_channel <- gold_message
}

func (man * Manager) notify_new_plant(plant_message PlayerPlantMessage) {
	man.plant_message_channel <- plant_message
}


/*
 Sends a ping message to all connected clients
*/
func (man *Manager) broadcast_ping() {
	for _, value := range man.all_connections {
		value.ping()
	}
}

func (man *Manager) broadcast_msg(msg string) {
	for _, value := range man.all_connections {
		value.send_msg(msg)
	}
}

func (man *Manager) add_msg_for_broadcast(msg string) {
	// FIXME: in retrospect, probably should have just used
	// broadcast channel for all types of messages.
	man.broadcast_channel <- msg
}

func (man * Manager) handle_player_gold_message(player_gold_message  PlayerGoldMessage) {
	
	if (player_gold_message.GoldMsgType == PLAYER_GOLD_MESSAGE_TYPE_GRAB) {
		log.Println("Received request to grab gold");

		player_id := player_gold_message.ID
		player, exists := man.all_connections[player_id]
		if (! exists) {
			// player has since disconnected
			return
		}

		amt_to_grab := player_gold_message.Amt
		man.gold_manager.grab_gold(player,amt_to_grab,player.pos.make_copy())
		
	} else if (player_gold_message.GoldMsgType == PLAYER_GOLD_MESSAGE_TYPE_DROP) {
		log.Println("Received request to drop gold")
		
		player_id := player_gold_message.ID
		player, exists := man.all_connections[player_id]
		if (! exists) {
			// player has disconnected already
			return
		}

		amt_to_grab := player_gold_message.Amt
		man.gold_manager.drop_gold(player,amt_to_grab,player.pos.make_copy())
		
	} else if (player_gold_message.GoldMsgType == PLAYER_GOLD_MESSAGE_TYPE_DEDUCT) {
		log.Println("Received request to deduct gold")
		
		// deduct is used to trade gold for player points
		player_id := player_gold_message.ID
		player, exists := man.all_connections[player_id]
		if (! exists) {
			// player has disconnected already
			return
		}
		player.buy_points(player_gold_message.Amt,man.gold_manager)
	} else if (player_gold_message.GoldMsgType == PLAYER_GOLD_MESSAGE_TYPE_ADD) {
		// add gold to existing stash if nearby, otherwise, create new stash
		log.Println("Received request to add gold")
		var position Position
		position.x = player_gold_message.X
		position.y = player_gold_message.Y
		position.z = player_gold_message.Z
		man.gold_manager.add_stash(player_gold_message.Amt, position) 
	} else {
		log.Println("Warning: unrecognized gold message type")
	}		
}


func (man *Manager) manager_loop() {
	// infinite loop waiting on additional work
	for {
		select {
			
		case player_to_register := <- man.register_channel:
			log.Println("Registering new player")
			man.all_connections[player_to_register.id] = player_to_register
			
		case player_to_unregister := <- man.unregister_channel:
			log.Println("Removing player")

			// remove map
			delete (man.all_connections, player_to_unregister.id)
			close (player_to_unregister.msg_output_queue)
			
			// construct the message to send to all others
			var plm PlayerLogoutMessage
			plm.MsgType = PLAYER_LOGOUT_MESSAGE_TYPE
			plm.ID = player_to_unregister.id
			byter, _ := json.Marshal(plm)
			msg_string := string(byter)

			for _, value := range man.all_connections {
				value.send_msg(msg_string)
			}
			
		case player_position_message := <- man.position_update_channel:
			byter, _ := json.Marshal(player_position_message)
			msg_string := string(byter)
			for key, value := range man.all_connections {
				if key != player_position_message.ID {
					// send the position on
					value.send_msg(msg_string)
				}
			}
		case player_gold_message := <- man.player_gold_message_channel:
			// Either adding more gold to the world,
			// grabbing gold, trading in existing gold, or dumping gold.
			log.Println("Received a request to grab surrounding gold")
			man.handle_player_gold_message(player_gold_message)

		case gold_message := <- man.gold_message_channel:
			log.Println("Sending gold message update to all players")
			byter, _ := json.Marshal(gold_message)
			msg_string := string(byter)
			for _, value := range man.all_connections {
				value.send_msg(msg_string)
			}
		case plant_message := <- man.plant_message_channel:
			byter, _ := json.Marshal(plant_message)
			msg_string := string(byter)
			for _, value := range man.all_connections {
				value.send_msg(msg_string)
			}

		case broadcast_message := <- man.broadcast_channel:
			man.broadcast_msg(broadcast_message)
		}
	}
}
