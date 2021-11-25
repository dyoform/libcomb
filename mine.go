package main

import (
	"crypto/sha256"
	"sync"
)

var commits_mutex sync.RWMutex

var commits map[[32]byte]utxotag
var combbases map[[32]byte]struct{}

var commit_cache_mutex sync.Mutex

var commit_currently_loaded utxotag
var commit_cache [][32]byte
var commit_tag_cache []utxotag
var commit_rollback [][32]byte
var commit_rollback_tags []utxotag

func init() {
	commits = make(map[[32]byte]utxotag)
	combbases = make(map[[32]byte]struct{})
}

func height_view() (h uint32) {
	commit_cache_mutex.Lock()
	h = commit_currently_loaded.height
	commit_cache_mutex.Unlock()
	return h
}


func miner_process_new() {
	for i := range commit_cache {

		if _, ok5 := commits[commit_cache[i]]; !ok5 {
			var basetag = commit_tag_cache[i]
			var btag = basetag

			var bheight = uint64(btag.height)

			segments_coinbase_mine(commit_cache[i], bheight)
			combbases[commit_cache[i]] = struct{}{}

			break
		}
	}
	for key, val := range commit_cache {
		if _, ok5 := commits[val]; ok5 {
		} else {
			//CommitDbWrite(val, hex2byte8(serializeutxotag(commit_tag_cache[key])))
			commits[val] = commit_tag_cache[key]
		}
	}

	for iter, key := range commit_cache {
		var tagval = commit_tag_cache[iter]

		merkle_mine(key)

		txleg_mutex.RLock()

		txlegs_each_leg_target(key, func(tx *[32]byte) bool {

			var oldactivity = tx_legs_activity[*tx]
			var newactivity = oldactivity
		outer:
			for i := uint(0); i < 21; i++ {
				if oldactivity&(1<<i) != 0 {
					continue
				}
				var roottag = tagval

				segments_transaction_mutex.RLock()

				var val = segments_transaction_data[*tx][i]

				segments_transaction_mutex.RUnlock()

				if key == commit(val[0:]) {

					var hash = val

					for j := 0; j < 65536; j++ {
						hash = sha256.Sum256(hash[0:])

						var candidaterawtag, ok3 = commits[commit(hash[0:])]

						if !ok3 {
							continue
						}
						var candidatetag = candidaterawtag

						if utag_cmp(&roottag, &candidatetag) >= 0 {

							continue outer
						}
					}
					newactivity |= (1 << i)
				}
			}

			if oldactivity != newactivity {
				tx_legs_activity[*tx] = newactivity
				if newactivity == 2097151 {

					logf("block confirms transaction %X \n", *tx)

					segments_transaction_mutex.RLock()

					var actuallyfrom = segments_transaction_data[*tx][21]

					segments_transaction_mutex.RUnlock()

					txdoublespends_each_doublespend_target(actuallyfrom, func(txidto *[2][32]byte) bool {
						if *tx == (*txidto)[0] {
							segments_transaction_next[actuallyfrom] = *txidto
							return false
						}
						return true
					})

					var maybecoinbase = commit(actuallyfrom[0:])
					if _, ok1 := combbases[maybecoinbase]; ok1 {
						segments_coinbase_trickle_auto(maybecoinbase, actuallyfrom)
					}

					segments_transaction_trickle(make(map[[32]byte]struct{}), actuallyfrom)
				}
			}

			return true
		})

		txleg_mutex.RUnlock()
	}

	commit_cache = nil
	commit_tag_cache = nil
}

func miner_process_rollback() {
	var unwritten bool = false
	var reorg_height uint64
	for i := range commit_rollback {
		if tagcommit, ok5 := commits[commit_rollback[i]]; ok5 {

			var basetag = commit_rollback_tags[i]
			var ctag = tagcommit
			var btag = basetag

			if utag_cmp(&ctag, &btag) != 0 {
				continue
			}

			var bheight = uint64(btag.height)

			if _, ok6 := combbases[commit_rollback[i]]; ok6 {

				segments_coinbase_unmine(commit_rollback[i], bheight)
				delete(combbases, commit_rollback[i])

			}

			break
		}
	}
	for i := len(commit_rollback) - 1; i >= 0; i-- {
		key := commit_rollback[i]

		if tagcommit, ok5 := commits[key]; !ok5 {
		} else {

			taggy := commit_rollback_tags[i]

			var ctag = tagcommit
			var btag = taggy

			if utag_cmp(&ctag, &btag) == 0 {
				//CommitDbUnWrite(key)
				delete(commits, key)
				unwritten = true

				if enable_used_key_feature {
					log("reorg commit height", ctag.height)

					reorg_height = uint64(ctag.height)

					used_key_commit_reorg(key, reorg_height)
				}
			}
		}
	}
	for _, key := range commit_rollback {

		if _, ok5 := commits[key]; ok5 {
			continue
		}

		merkle_unmine(key)

		txleg_mutex.RLock()

		txlegs_each_leg_target(key, func(tx *[32]byte) bool {

			var oldactivity = tx_legs_activity[*tx]
			var newactivity = oldactivity

			for i := uint(0); i < 21; i++ {
				if oldactivity&(1<<i) == 0 {
					continue
				}

				var val = segments_transaction_data[*tx][i]

				if key == commit(val[0:]) {
					newactivity &= 2097151 ^ (1 << i)
				}
			}

			if oldactivity != newactivity {
				tx_legs_activity[*tx] = newactivity

				if oldactivity == 2097151 {
					logf("block rollbacks transaction %X \n", *tx)

					var actuallyfrom = segments_transaction_data[*tx][21]

					segments_transaction_untrickle(nil, actuallyfrom, 0xffffffffffffffff)

					delete(segments_transaction_next, actuallyfrom)
				}
			}

			return true
		})

		txleg_mutex.RUnlock()
	}

	commit_rollback = nil
	commit_rollback_tags = nil

	if unwritten && enable_used_key_feature {
		log("reorg block height", reorg_height)
		used_key_height_reorg(reorg_height)
	}

	commit_currently_loaded.height--;
}

func miner_process() {
	commit_cache_mutex.Lock()
	commits_mutex.Lock()
	if len(commit_rollback) > 0 && len(commit_cache) > 0 {
		//protect from this in the interface!
		commits_mutex.Unlock()
		commit_cache_mutex.Unlock()
		return
	} else if len(commit_cache) > 0 {
		miner_process_new()
	} else if len(commit_rollback) > 0 {
		miner_process_rollback()
	} else {
		//nothing to do!
	}

	resetgraph()

	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
	return
}

func miner_add_commit(rawcommit [32]byte, tag utxotag) {
	var direction_mine_unmine = utag_mining_sign(tag)
	var is_coinbase bool

	commit_cache_mutex.Lock()

	is_coinbase = len(commit_cache)+len(commit_rollback) == 0

	if is_coinbase && direction_mine_unmine == UTAG_MINE && commit_currently_loaded.height >= tag.height {
		commit_cache_mutex.Unlock()
		logf("error: mined first commitment must be on greater height\n")
		return
	}
	if is_coinbase && direction_mine_unmine == UTAG_UNMINE && commit_currently_loaded.height < tag.height {
		commit_cache_mutex.Unlock()
		logf("error: unmined first commitment must be on smaller height\n")
		return
	}
	if !is_coinbase && commit_currently_loaded.height != tag.height {
		commit_cache_mutex.Unlock()
		logf("error: commitment must be on same height as first commitment\n")
		return
	}

	//must have patched in outnum and txnum?
	if tag.height >= strictly_monotonic_vouts_bugfix_fork_height {
		if direction_mine_unmine == UTAG_UNMINE {
			tag.outnum = uint16(len(commit_rollback) % 10000)
			tag.txnum = uint16(len(commit_rollback) / 10000)
		} else if direction_mine_unmine == UTAG_MINE {
			tag.outnum = uint16(len(commit_cache) % 10000)
			tag.txnum = uint16(len(commit_cache) / 10000)
		}
	}

	if direction_mine_unmine == UTAG_UNMINE {
		commit_rollback = append(commit_rollback, rawcommit)
		commit_rollback_tags = append(commit_rollback_tags, tag)
	} else if direction_mine_unmine == UTAG_MINE {
		commit_cache = append(commit_cache, rawcommit)
		commit_tag_cache = append(commit_tag_cache, tag)
	}

	commits_mutex.Lock()
	commit_currently_loaded = tag
	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
}