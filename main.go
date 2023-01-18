package main

import (
	"fmt"
	"math/rand"
	"time"
)

func generateZipfData(s, v float64, imax uint64, N int64) []uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	zipf := rand.NewZipf(r, s, v, imax)
	data := make([]uint64, 0, N)
	for i := int64(0); i < N; i++ {
		data = append(data, zipf.Uint64())
	}
	return data
}

func exactNDV(data []uint64) int {
	h := make(map[uint64]struct{}, 1000)
	for _, x := range data {
		h[x] = struct{}{}
	}
	return len(h)
}

func sampleData(data []uint64, rate float64) []uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	samples := make([]uint64, int(float64(len(data))*rate))
	for _, x := range data {
		if r.Float64() < rate {
			samples = append(samples, x)
		}
	}
	return samples
}

func firstOrderJackknifeEstimator(samples []uint64) float64 {
	h := make(map[uint64]uint64, 1000)
	for _, x := range samples {
		h[x] = h[x] + 1
	}
	n := float64(len(samples))
	observedNDV := float64(len(h))
	f1 := 0.0
	for _, frequency := range h {
		if frequency == 1 {
			f1 += 1
		}
	}
	estimatedNDV := observedNDV + (n-1)/n*f1
	return estimatedNDV
}

func main() {
	s := 1.5
	v := 1.0
	imax := uint64(1000000000)
	N := int64(10000000)
	data := generateZipfData(s, v, imax, N)
	ndv := exactNDV(data)
	fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], N:%v, NDV:%v\n", s, v, imax, N, ndv)
	sampleRate := 0.8
	samples := sampleData(data, sampleRate)
	estimatedNDV := firstOrderJackknifeEstimator(samples)
	fmt.Printf("sample rate: %v, estimated NDV from first-order Jackknife Estimator:%v", sampleRate, estimatedNDV)
}
