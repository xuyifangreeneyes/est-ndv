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

// https://mmeredith.net/blog/2013/1312_Jackknife_estimators.htm
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

// https://mmeredith.net/blog/2013/1312_Jackknife_estimators.htm
func secondOrderJackknifeEstimator(samples []uint64) float64 {
	h := make(map[uint64]uint64, 1000)
	for _, x := range samples {
		h[x] = h[x] + 1
	}
	n := float64(len(samples))
	observedNDV := float64(len(h))
	f1, f2 := 0.0, 0.0
	for _, frequency := range h {
		if frequency == 1 {
			f1 += 1
		} else if frequency == 2 {
			f2 += 1
		}
	}
	estimatedNDV := observedNDV + (2*n-3)/n*f1 - (n-2)*(n-2)/n/(n-1)*f2
	return estimatedNDV
}

// https://github.com/postgres/postgres/blob/master/src/backend/commands/analyze.c#L2210-L2252
func duj1Estimator(samples []uint64, N int64) float64 {
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
	estimatedNDV := n * observedNDV / (n - f1 + f1*n/float64(N))
	return estimatedNDV
}

func qerror(act, est float64) float64 {
	if act > est {
		return act / est
	}
	return est / act
}

func main() {
	s := 1.5
	v := 1.0
	imax := uint64(10000000000)
	NList := []int64{1e5, 1e7, 1e8}
	for _, N := range NList {
		data := generateZipfData(s, v, imax, N)
		ndv := exactNDV(data)
		fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], N:%v, NDV:%v\n", s, v, imax, N, ndv)
		sampleRateList := []float64{0.5, 1e-1, 1e-2, 1e-3, 1e-4}
		for _, sampleRate := range sampleRateList {
			samples := sampleData(data, sampleRate)
			estNDV1 := firstOrderJackknifeEstimator(samples)
			fmt.Printf("sample rate: %v, first-order Jackknife NDV:%v, q-error:%v\n", sampleRate, estNDV1, qerror(float64(ndv), estNDV1))
			estNDV2 := secondOrderJackknifeEstimator(samples)
			fmt.Printf("sample rate: %v, second-order Jackknife NDV:%v, q-error:%v\n", sampleRate, estNDV2, qerror(float64(ndv), estNDV2))
			estNDV3 := duj1Estimator(samples, N)
			fmt.Printf("sample rate: %v, Duj1 NDV:%v, q-error:%v\n", sampleRate, estNDV3, qerror(float64(ndv), estNDV3))
		}
	}
}
