// The code is mainly from https://github.com/eclesh/hyperloglog/blob/master/hyperloglog.go
package main

import (
	"fmt"
	"math"
)

var (
	exp32 = math.Pow(2, 32)
)

type HyperLogLog struct {
	m         uint32  // Number of registers
	b         uint8   // Number of bits used to determine register index
	alpha     float64 // Bias correction constant
	registers []uint8
}

// getAlpha computes bias correction alpha_m.
func getAlpha(m uint32) (result float64) {
	switch m {
	case 16:
		result = 0.673
	case 32:
		result = 0.697
	case 64:
		result = 0.709
	default:
		result = 0.7213 / (1.0 + 1.079/float64(m))
	}
	return result
}

// NewHyperLogLog returns a new HyperLogLog with the given number of registers. More registers leads to lower error in
// your estimated count, at the expense of memory.
//
// Choose a power of two number of registers, depending on the amount of memory you're willing to use and the error
// you're willing to tolerate. Each register uses one byte of memory.
//
// Approximate error will be: 1.04 / sqrt(registers)
func NewHyperLogLog(registers uint32) (*HyperLogLog, error) {
	if (registers & (registers - 1)) != 0 {
		return nil, fmt.Errorf("number of registers %d not a power of two", registers)
	}
	h := &HyperLogLog{}
	h.m = registers
	h.b = uint8(math.Ceil(math.Log2(float64(registers))))
	h.alpha = getAlpha(registers)
	h.Reset()
	return h, nil
}

// Reset sets all registers to zero.
func (h *HyperLogLog) Reset() {
	h.registers = make([]uint8, h.m)
}

// rho calculates the position of the leftmost 1-bit.
func rho(val uint64, max uint8) uint8 {
	r := uint8(1)
	for val&0x8000000000000000 == 0 && r <= max {
		r++
		val <<= 1
	}
	return r
}

// InsertHash inserts val into HyperLogLog. val should be a 64-bit unsigned integer from a good hash function.
func (h *HyperLogLog) InsertHash(val uint64) {
	k := 64 - h.b
	r := rho(val<<h.b, k)
	j := val >> k
	if r > h.registers[j] {
		h.registers[j] = r
	}
}

// Count returns the estimated NDV.
func (h *HyperLogLog) Count() uint64 {
	sum := 0.0
	m := float64(h.m)
	for _, val := range h.registers {
		sum += 1.0 / math.Pow(2.0, float64(val))
	}
	estimate := h.alpha * m * m / sum
	if estimate <= 2.5*m {
		// Small range correction
		zeros := 0
		for _, r := range h.registers {
			if r == 0 {
				zeros++
			}
		}
		if zeros > 0 {
			estimate = m * math.Log(m/float64(zeros))
		}
	}
	return uint64(estimate + 0.5)
}

// Merge merges another HyperLogLog into this one. The number of registers in each must be the same.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h.m != other.m {
		return fmt.Errorf("number of registers doesn't match: %d != %d", h.m, other.m)
	}
	for j, r := range other.registers {
		if r > h.registers[j] {
			h.registers[j] = r
		}
	}
	return nil
}
