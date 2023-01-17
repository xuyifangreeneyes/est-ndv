package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := 1.5
	v := 1.0
	imax := uint64(1000000000)
	zipf := rand.NewZipf(r, s, v, imax)
	n := 1000000
	data := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		data = append(data, zipf.Uint64())
	}
	h := make(map[uint64]struct{}, 1000)
	for _, x := range data {
		h[x] = struct{}{}
	}
	ndv := len(h)
	fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], n:%v, ndv:%v\n", s, v, imax, n, ndv)
}
