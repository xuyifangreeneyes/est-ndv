package main

// FirstOrderJackknifeEstimator is from https://mmeredith.net/blog/2013/1312_Jackknife_estimators.htm
func FirstOrderJackknifeEstimator(samples []uint64, N int64) float64 {
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

// SecondOrderJackknifeEstimator is from https://mmeredith.net/blog/2013/1312_Jackknife_estimators.htm
func SecondOrderJackknifeEstimator(samples []uint64, N int64) float64 {
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

// Duj1Estimator is from https://github.com/postgres/postgres/blob/master/src/backend/commands/analyze.c#L2210-L2252
func Duj1Estimator(samples []uint64, N int64) float64 {
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
