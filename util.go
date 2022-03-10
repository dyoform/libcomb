package libcomb

import (
	"crypto/sha256"
	"hash"
)

var whitepaper = [32]byte{106, 251, 172, 89, 92, 29, 7, 163, 212, 197, 23, 151, 88, 245, 188, 228, 70, 42, 108, 38, 63, 110, 109, 252, 217, 66, 1, 20, 51, 173, 170, 231}
var testnet_whitepaper = [32]byte{46, 56, 65, 182, 231, 94, 151, 23, 171, 125, 42, 139, 87, 36, 139, 127, 97, 26, 84, 115, 56, 27, 94, 67, 42, 175, 143, 232, 136, 116, 251, 254}

func Hash256(data []byte) (out [32]byte) {
	if !mode.testnet {
		return sha256.Sum256(data)
	} else {
		var h hash.Hash = sha256.New()
		h.Write(testnet_whitepaper[:])
		h.Write(testnet_whitepaper[:])
		h.Write(data)
		h.Sum(out[0:0])
		return out
	}
}

func Hash256Concat32(data [][32]byte) (out [32]byte) {
	var c []byte = make([]byte, 32*len(data))
	for i, d := range data {
		copy(c[i*32:(i+1)*32], d[:])
	}
	return Hash256(c[:])
}

func Hash256Adjacent(a [32]byte, b [32]byte) (out [32]byte) {
	var c [64]byte
	copy(c[0:], a[:])
	copy(c[32:], b[:])
	return Hash256(c[:])
}

const precision = 50

func mult128to128(o1hi uint64, o1lo uint64, o2hi uint64, o2lo uint64, hi *uint64, lo *uint64) {
	mult64to128(o1lo, o2lo, hi, lo)

	*hi += o1hi * o2lo
	*hi += o2hi * o1lo
}

func mult64to128(op1 uint64, op2 uint64, hi *uint64, lo *uint64) {
	var u1 = (op1 & 0xffffffff)
	var v1 = (op2 & 0xffffffff)
	var t = (u1 * v1)
	var w3 = (t & 0xffffffff)
	var k = (t >> 32)

	op1 >>= 32
	t = (op1 * v1) + k
	k = (t & 0xffffffff)
	var w1 = (t >> 32)

	op2 >>= 32
	t = (u1 * op2) + k
	k = (t >> 32)

	*hi = (op1 * op2) + w1 + k
	*lo = (t << 32) + w3
}

func log2(xx uint64) (uint64, uint64) {
	var b uint64 = 1 << (precision - 1)
	var yhi uint64 = 0
	var ylo uint64 = 0
	var zhi uint64 = xx >> (64 - precision)
	var zlo uint64 = xx << precision

	for (zhi > 0) || (zlo >= 2<<precision) {
		zlo = (zhi << (64 - 1)) | (zlo >> 1)
		zhi = zhi >> 1
		if ylo+(1<<precision) < ylo {
			yhi++
		}

		ylo += 1 << precision
	}

	for i := 0; i < precision; i++ {

		mult128to128(zhi, zlo, zhi, zlo, &zhi, &zlo)

		zlo = (zhi << (64 - precision)) | (zlo >> precision)
		zhi = zhi >> precision

		if (zhi > 0) || (zlo >= 2<<precision) {

			zlo = (zhi << (64 - 1)) | (zlo >> 1)
			zhi = zhi >> 1

			if ylo+b < ylo {
				yhi++
			}

			ylo += b
		}
		b >>= 1
	}

	return yhi, ylo
}
