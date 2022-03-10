package libcomb

var commits map[[32]byte]Tag
var height uint64

func commits_initialize() {
	commits = make(map[[32]byte]Tag)
	height = 0
}

func load_block(b Block) {
	height++
	var ok bool
	var tag Tag = Tag{height, 0}
	for _, commit := range b.Commits {
		//skip seen commits
		if _, ok = commits[commit]; ok {
			continue
		}

		//store commit
		commits[commit] = tag

		//check the first commit for a coinbase (there will be one!)
		if tag.Order == 0 {
			coinbase_check_commit(commit)
		}

		//trigger any constructs related to this commit
		constructs_check_commit(commit)

		tag.Order++
	}
}

func unload_block() uint64 {
	//remove all commits at the current height
	for c, tag := range commits {
		if tag.Height == height {
			delete(commits, c)
		}
	}
	//decrement the height
	height--
	return height
	//note: balance graph is now invalid, needs to be reconstructed
}

func commit(hash [32]byte) [32]byte {
	var data = [2][32]byte{whitepaper, hash}
	return Hash256Concat32(data[:])
}
