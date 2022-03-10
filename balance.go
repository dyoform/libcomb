package libcomb

var balance map[[32]byte]uint64
var balance_edge map[[32]byte][32]byte    //source -> destination
var balance_one_off map[[32]byte][32]byte //source -> destination
var balance_coinbases map[[32]byte]uint64 //commit -> reward

func balance_initialize() {
	balance = make(map[[32]byte]uint64)
	balance_edge = make(map[[32]byte][32]byte)
	balance_one_off = make(map[[32]byte][32]byte)
	balance_coinbases = make(map[[32]byte]uint64)
}

func balance_propagate(address [32]byte) {
	var visited map[[32]byte]struct{} = make(map[[32]byte]struct{})
	visited[address] = struct{}{}
	for {
		if next, ok := balance_edge[address]; ok {
			if _, ok = visited[next]; ok {
				balance[address] = 0
				return
			}

			visited[next] = struct{}{}
			balance[next] += balance[address]
			balance[address] = 0
			address = next
		} else {
			break
		}
	}
	constructs_trigger(address)
}

func balance_stack_redirect(source, destination, change [32]byte, sum uint64) {
	balance[source] -= sum
	balance[destination] += sum
	balance_one_off[source] = destination
	balance_edge[source] = change
	balance_propagate(destination) //destination is transfered first (could be significant?)
	balance_propagate(source)
}

func balance_redirect(source, destination [32]byte) {
	balance_edge[source] = destination
	balance_propagate(source)
}

func balance_rebuild() {
	balance_initialize()           //reset balance graph
	for _, c := range constructs { //trigger all constructs
		c.trigger()
	}
}
