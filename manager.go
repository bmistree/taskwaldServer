package main

import "log"


/***************** MANAGER DEFINITIONS ********/

type Manager struct {
	// map [<key type>] <value type>
	all_connections map[uint32] *Player
	register_channel chan *Player
	unregister_channel chan *Player
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
			man.broadcast_ping()
		case player_to_unregister := <- man.unregister_channel:
			log.Println("Removing player")
			delete (man.all_connections, player_to_unregister.id)
			close (player_to_unregister.msg_output_queue)
		}
	}
}
