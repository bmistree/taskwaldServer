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
}

func (man * Manager) receive_position_update(ppm PlayerPositionMessage) {
	// only send update messages to those players that are within
	man.position_update_channel <- ppm
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
			log.Println("Received position update message")
			byter, _ := json.Marshal(player_position_message)
			msg_string := string(byter)
			for key, value := range man.all_connections {
				if key != player_position_message.ID {
					// send the position on
					value.send_msg(msg_string)
				}
			}
		}
	}
}
