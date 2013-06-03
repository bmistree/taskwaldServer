package main
import "log"
import "code.google.com/p/go.net/websocket"
import "net/http"

var LISTENING_ADDR = "127.0.0.1:18080"


// FIXME: want this to be atomic
var id_counter uint32 = 0

var manager_singleton = Manager{
	all_connections: make (map [uint32] *Player),
	register_channel: make (chan *Player),
        unregister_channel: make (chan *Player)}


/**
  * When receive a new web socket connection, run this code:
  *    1) Creates a new Player
  *    2) Notifies the manager to include this connection in map

  *    3) Starts spinning on channels waiting for sending messages and
       receiving messages from player
 */
func ws_registration_handler(conn *websocket.Conn) {
	id_counter += 1  // FIXME: make this atomic
	player := &Player{id: id_counter,msg_output_queue: make(chan string, 256), conn: conn}

	// listen for messages from server to client
	go player.write_msg_loop()	
	manager_singleton.register_channel <- player

	// When this function completes, remove player from manager
	defer func() {
		manager_singleton.unregister_channel <- player
	}()
	
	// do not call as separate go routine: as soon as handler exits, connection closes
	player.read_msg_loop()
}


func main(){
	// pos := Position {0,0,0}
	// pos.x = pos.y;
	// Start listening at address

	log.Println("Listening for connections");

	go manager_singleton.manager_loop()
	http.Handle("/ws", websocket.Handler(ws_registration_handler))
	if err := http.ListenAndServe(LISTENING_ADDR, nil); err != nil {
	 	log.Fatal("ListenAndServe:", err)
	}
}

