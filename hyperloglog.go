// The code is mainly from https://github.com/eclesh/hyperloglog/blob/master/hyperloglog.go
package main

import (
	"encoding/binary"
	"fmt"
	"hash"
	"math"

	"github.com/twmb/murmur3"
)

type HyperLogLog struct {
	m         uint32  // Number of registers
	b         uint8   // Number of bits used to determine register index
	alpha     float64 // Bias correction constant
	registers []uint8
	hashFunc  hash.Hash64
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
	h.hashFunc = murmur3.New64()
	return h, nil
}

// GetRegisterNum returns the number of registers.
func (h *HyperLogLog) GetRegisterNum() uint32 {
	return h.m
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

// InsertUint64 inserts an uint64 number into HyperLogLog.
func (h *HyperLogLog) InsertUint64(x uint64) error {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, x)
	return h.InsertValue(b)
}

// InsertValue inserts value into HyperLogLog.
func (h *HyperLogLog) InsertValue(value []byte) error {
	h.hashFunc.Reset()
	_, err := h.hashFunc.Write(value)
	if err != nil {
		return err
	}
	hashVal := h.hashFunc.Sum64()
	k := 64 - h.b
	r := rho(hashVal<<h.b, k)
	j := hashVal >> k
	if r > h.registers[j] {
		h.registers[j] = r
	}
	return nil
}

// Count returns the estimated NDV.
func (h *HyperLogLog) Count() float64 {
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
	return estimate
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

func (h *HyperLogLog) Clone() *HyperLogLog {
	registers := make([]uint8, h.m)
	copy(registers, h.registers)
	return &HyperLogLog{
		m:         h.m,
		b:         h.b,
		alpha:     h.alpha,
		registers: registers,
		hashFunc:  murmur3.New64(),
	}
}
