package libcomb

import (
	"fmt"
	"log"
)

type UnsignedMerkleSegment struct {
	Tips [2][32]byte
	Next [32]byte
	Root [32]byte
}

func (m UnsignedMerkleSegment) ID() [32]byte {
	return Hash256Adjacent(Hash256Concat32([][32]byte{m.Next, m.Tips[0], m.Tips[1]}), m.Root)
}

func (m UnsignedMerkleSegment) trigger() (err error) {
	return nil
}

func (m UnsignedMerkleSegment) triggers() [][32]byte {
	return nil
}

func unsigned_merkle_segment_lookup(id [32]byte) (UnsignedMerkleSegment, error) {
	var u UnsignedMerkleSegment
	if c, ok := constructs[id]; ok {
		if u, ok = c.(UnsignedMerkleSegment); !ok {
			return u, fmt.Errorf("not an unsigned merkle segment")
		}
	} else {
		return u, fmt.Errorf("not a construct")
	}

	return u, nil
}

type MerkleSegment struct {
	Tips      [2][32]byte
	Signature [2][32]byte
	Branches  [16][32]byte
	Leaf      [32]byte
	Next      [32]byte

	Root [32]byte
	id   [32]byte
}

func (m MerkleSegment) ID() [32]byte {
	if m.id == [32]byte{} {
		log.Panicf("merkle segment has not been recovered")
	}
	return m.id
}

func (m MerkleSegment) Active() bool {
	if _, ok := balance_edge[m.id]; ok {
		return true
	}
	return false
}

func (m MerkleSegment) trigger() (err error) {
	if m.Active() {
		return
	}

	var signature_commits [2][32]byte
	signature_commits[0] = commit(m.Signature[0])
	signature_commits[1] = commit(m.Signature[1])

	//check signature is committed
	if missing, ok := query_commits_exist(signature_commits[:]); !ok {
		return fmt.Errorf("missing %d signature commits", len(missing))
	}

	//check leg for older signatures
	for i := range m.Signature {
		var leg_commits [][32]byte = make([][32]byte, 0)
		var hash = m.Signature[i]
		for hash != m.Tips[i] {
			hash = Hash256(hash[:])
			leg_commits = append(leg_commits, commit(hash))
		}
		if query_commits_any_older_than(leg_commits, signature_commits[i]) {
			return fmt.Errorf("older signature on leg %d", i)
		}
	}

	balance_redirect(m.ID(), m.Leaf)
	fmt.Println("merkle activated")
	return nil
}

func (m MerkleSegment) triggers() (t [][32]byte) {
	for _, s := range m.Signature {
		t = append(t, commit(s))
	}
	return t
}

func merkle_traverse_tree(tree [65536][32]byte, number uint16) (root [32]byte, branches [16][32]byte, leaf [32]byte) {
	leaf = tree[number]
	for j := 0; j < 16; j++ {
		branches[j] = tree[number^1]
		for i := 0; i < 1<<uint(15-j); i++ {
			tree[i] = Hash256Adjacent(tree[2*i], tree[2*i+1])
		}
		number >>= 1
	}
	root = tree[0]
	return root, branches, leaf
}

func merkle_recover(m *MerkleSegment) (err error) {
	var number uint16
	var link [32]byte

	//recover number from signature and tips
	if number, err = merkle_recover_number(m); err != nil {
		return err
	}

	//recover merkle root from leaf + branches + number
	m.Root = m.Leaf
	for i := byte(0); i < 16; i++ {
		if ((number >> i) & 1) == 0 {
			m.Root = Hash256Adjacent(m.Root, m.Branches[i])
		} else {
			m.Root = Hash256Adjacent(m.Branches[i], m.Root)
		}
	}

	//recover decider link from tips + next
	var data [3][32]byte = [3][32]byte{m.Next, m.Tips[0], m.Tips[1]}
	link = Hash256Concat32(data[:])

	m.id = Hash256Adjacent(link, m.Root)

	return nil
}

func merkle_recover_number(m *MerkleSegment) (number uint16, err error) {
	var hash = m.Signature[1]
	for i := uint16(0); i < 65535; i++ {
		if hash == m.Tips[1] {
			number = i
			break
		}
		hash = Hash256(hash[0:])
	}

	if hash != m.Tips[1] {
		return 0, fmt.Errorf("invalid signature")
	}

	hash = m.Signature[0]
	for i := uint16(0); i < uint16(65535-number); i++ {
		hash = Hash256(hash[:])
	}

	if hash != m.Tips[0] {
		return 0, fmt.Errorf("mismatched signature")
	}

	return number, nil
}

func merkle_compare(a MerkleSegment, b MerkleSegment) bool {
	if a.ID() != b.ID() {
		return false
	}
	//ID doesnt uniquely identify a signed Merkle Segment, need to check destination etc

	var out bool = true
	out = out && (a.Leaf == b.Leaf)
	out = out && (a.Branches == b.Branches)
	out = out && (a.Signature == b.Signature)

	return out
}
