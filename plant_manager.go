package main

import "sync"

// Tracks all of the plants in the world.

type PlantId uint32
	
type PlantManagerSingleton struct {
	plant_id PlantId
	// from stash_id
	all_plants map[PlantId] *Plant
	connection_manager * Manager
	lock sync.Mutex
}

func (pm * PlantManagerSingleton) acquire_lock() {
	pm.lock.Lock()
}
func (pm * PlantManagerSingleton) release_lock() {
	pm.lock.Unlock()
}


func (pm * PlantManagerSingleton) add_plant(player_plant_message PlayerPlantMessage) {
	var pos Position
	pos.x = player_plant_message.X
	pos.y = player_plant_message.Y
	pos.z = player_plant_message.Z
	
	pm.acquire_lock()
	plant := new (Plant)
	plant.plant_id = pm.plant_id
	pm.plant_id ++
	plant.pos = pos
	pm.all_plants[plant.plant_id] = plant
	pm.release_lock()

	pm.connection_manager.notify_new_plant(player_plant_message)
}


type Plant struct {
	plant_id PlantId
	pos Position
}