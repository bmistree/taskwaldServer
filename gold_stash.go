package main

type GoldStash struct {
	stash_id GoldStashId
	amt uint32
	pos Position
}


func (gs * GoldStash) add_gold(amt uint32) {
	gs.amt += amt
}

// Try to acquire some of the gold.  If amt is 0, then acquire all of
// it.
// @param {*bool} finished --- Whether the gold stash is
// exhausted (and should be removed).
// @returns {uint32} amount of gold acquired by user
func (gs * GoldStash) grab_gold(amt uint32) (uint32, bool) {
	var amt_acquired uint32
	amt_acquired = amt
	finished := true
	if ((amt == 0) || (amt > gs.amt)) {
		amt_acquired = gs.amt
		gs.amt = 0
	} else {
		finished = false
		gs.amt -= amt
	}
	return amt_acquired, finished
}