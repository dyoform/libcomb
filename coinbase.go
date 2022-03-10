package libcomb

import "fmt"

func coinbase_give_reward(c [32]byte) bool {
	if _, ok := balance_coinbases[c]; ok {
		fmt.Printf("already rewarded\n")
		return false //already awarded
	}

	var reward uint64
	if reward = query_get_coinbase(c); reward == 0 {
		fmt.Printf("not a coinbase\n")
		return false //not a coinbase
	}

	fmt.Printf("rewarded\n")
	balance_coinbases[c] = reward
	balance[c] += reward
	return true
}

func coinbase_check_commit(c [32]byte) {
	if a, ok := construct_uncommits[c]; ok {
		if !coinbase_give_reward(c) {
			return //no coinbase awarded
		}
		//redirect funds to the address
		balance_redirect(c, a)
	}
}

func coinbase_check_address(a [32]byte) {
	var c [32]byte = commit(a)

	if !coinbase_give_reward(c) {
		return //no coinbase awarded
	}

	//now propagate the coinbase to the construct
	balance_redirect(c, a)
}

func coin_supply(height uint64) (uint64, uint64) {
	var loghi, loglo = log2(height)

	var hi, lo uint64

	mult128to128(loghi, loglo, loghi, loglo, &hi, &lo)

	lo = lo>>(precision) | hi<<(64-precision)
	hi = hi >> (precision)

	var hi2, lo2 uint64

	mult128to128(hi, lo, hi, lo, &hi2, &lo2)

	lo2 = lo2>>(precision) | hi2<<(64-precision)
	hi2 = hi2 >> (precision)

	var hi3, lo3 uint64

	mult128to128(hi, lo, hi2, lo2, &hi3, &lo3)

	lo3 = lo3>>(precision) | hi3<<(64-precision)
	hi3 = hi3 >> (precision)

	lo3 = lo3>>(precision) | hi3<<(64-precision)
	hi3 = hi3 >> (precision)

	return lo3, loglo
}

func coinbase_reward(height uint64) uint64 {
	if height >= 21835313 {
		return 0
	}

	var decrease, _ = coin_supply(height)

	return 210000000 - decrease
}
