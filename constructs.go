package libcomb

import (
	"fmt"
	"log"
)

type Construct interface {
	ID() [32]byte
	trigger() error
	triggers() [][32]byte
}

var constructs map[[32]byte]Construct         //id -> construct
var construct_triggers map[[32]byte][32]byte  //trigger -> id
var construct_uncommits map[[32]byte][32]byte //commit(id) -> id
var construct_load_order [][32]byte           //golangs maps dont have a stable order

func constructs_initialize() {
	constructs = make(map[[32]byte]Construct)
	construct_uncommits = make(map[[32]byte][32]byte)
	construct_triggers = make(map[[32]byte][32]byte)
	construct_load_order = make([][32]byte, 0)
}

func constructs_load(c Construct) [32]byte {
	var id [32]byte = c.ID()

	if _, ok := constructs[id]; !ok {
		construct_load_order = append(construct_load_order, id)
	}

	coinbase_check_address(id)

	construct_uncommits[commit(id)] = id
	constructs[id] = c
	for _, t := range c.triggers() {
		construct_triggers[t] = id
	}
	if err := c.trigger(); err != nil {
		fmt.Println(err.Error())
	}

	return id
}

func constructs_check_commit(commit [32]byte) {
	var ok bool
	var id [32]byte

	//trigger any constructs that set 'commit' as a trigger
	if id, ok = construct_triggers[commit]; ok {
		if _, ok = constructs[id]; ok {
			constructs[id].trigger()
		} else {
			log.Panicf("commit triggers an unloaded construct (%X triggers %X)", commit, id)
		}
	}
}

func constructs_trigger(address [32]byte) bool {
	//lookup construct via 'address' and trigger it
	if _, ok := constructs[address]; ok {
		constructs[address].trigger()
		return true
	}
	return false
}
