package main

import "sync"

// Tracks all the individual stashes of gold in the world.  Players
// track their own gold themselves.

type GoldStashId uint32
	
type GoldManagerSingleton struct {
	stash_id GoldStashId
	// from stash_id
	all_stashes map[GoldStashId] *GoldStash
	connection_manager * Manager
	lock sync.Mutex
}

const GRAB_RADIUS = 3
const ADD_RADIUS = 1

func (gm * GoldManagerSingleton) acquire_lock() {
	gm.lock.Lock()
}
func (gm * GoldManagerSingleton) release_lock() {
	gm.lock.Unlock()
}


func (gm * GoldManagerSingleton) get_stashes_within_radius(pos Position, radius float64) []* GoldStash {
	var nearby_stashes [] * GoldStash
	
	radius_squared := radius * radius
	for _, gold_stash := range gm.all_stashes {
		delta_x := pos.x - gold_stash.pos.x
		delta_y := pos.y - gold_stash.pos.y
		delta_z := pos.z - gold_stash.pos.z
		dist_squared := delta_x * delta_x + delta_y * delta_y + delta_z * delta_z
		
		if (dist_squared < radius_squared) {
			nearby_stashes = append(nearby_stashes,gold_stash)
		}
	}
	return nearby_stashes
}

type AddedGoldSubmessage struct {
	stash_id GoldStashId
	amt uint32
	pos Position
}

type DeletedGoldSubmessage struct {
	stash_id GoldStashId
}
type ChangedGoldSubmessage struct {
	stash_id GoldStashId
	final_amt uint32
}
	

type GoldMessage struct {
	AddedSubmessages [] AddedGoldSubmessage
	DeletedSubmessages [] DeletedGoldSubmessage
	ChangedSubmessages [] ChangedGoldSubmessage
}


func (gm * GoldManagerSingleton) grab_gold(player * Player, amt uint32, pos Position, radius float64) uint32 {
	gm.acquire_lock()
	nearby_stashes := gm.get_stashes_within_radius(pos, GRAB_RADIUS)

	gold_message := new (GoldMessage)

	var stashes_to_delete [] GoldStashId
	var total_grabbed uint32
	total_grabbed = 0
	for _, gold_stash := range nearby_stashes {
		amt_grabbed, stash_finished := gold_stash.grab_gold(amt,)
		amt -= amt_grabbed
		total_grabbed += amt_grabbed
		if (stash_finished) {
			stashes_to_delete = append(stashes_to_delete,gold_stash.stash_id)
		} else {
			changed_submessage := ChangedGoldSubmessage {
				stash_id: gold_stash.stash_id,
			        final_amt: gold_stash.amt }
			gold_message.ChangedSubmessages = append(gold_message.ChangedSubmessages, changed_submessage)
		}
			
		// no more to grab
		if (amt == 0) {
			break
		}
	}

	for _, stash_id := range stashes_to_delete {
		deleted_submessage := DeletedGoldSubmessage  { stash_id: stash_id}
		gold_message.DeletedSubmessages = append(gold_message.DeletedSubmessages,deleted_submessage)
		delete (gm.all_stashes, stash_id)	
	}

	gm.connection_manager.gold_message_channel <- gold_message
	player.gold += total_grabbed
	gm.release_lock()
	return total_grabbed
}

func (gm * GoldManagerSingleton) add_stash(amt uint32, pos Position) {
	gm.acquire_lock()
	nearby_stashes := gm.get_stashes_within_radius(pos, ADD_RADIUS)
	if (len(nearby_stashes) != 0) {
		gold_stash := nearby_stashes[0]
		gold_stash.add_gold(amt)
	} else {
		gm.stash_id += 1
		new_stash := new (GoldStash)
		new_stash.stash_id = gm.stash_id
		new_stash.amt = amt
		new_stash.pos = pos	
	}
	gm.release_lock()
}