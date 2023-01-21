package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
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
func firstOrderJackknifeEstimator(samples []uint64, N int64) float64 {
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
func secondOrderJackknifeEstimator(samples []uint64, N int64) float64 {
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

func benchSampleBasedEstimators() {
	s := 1.5
	v := 1.0
	imax := uint64(10000000000)
	NList := []int64{1e6, 1e7, 1e8}
	bars := make([]components.Charter, 0, 3)
	for _, N := range NList {
		data := generateZipfData(s, v, imax, N)
		actNDV := exactNDV(data)
		fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], N:%v, NDV:%v\n", s, v, imax, N, actNDV)
		sampleRateList := []float64{0.5, 1e-1, 1e-2, 1e-3, 1e-4}
		results := make([][]float64, 3)
		for i := 0; i < 3; i++ {
			results[i] = make([]float64, len(sampleRateList))
		}
		estimatorNames := []string{"first-order Jackknife", "second-order Jackknife", "Duj1"}
		estimators := []func(samples []uint64, N int64) float64{firstOrderJackknifeEstimator, secondOrderJackknifeEstimator, duj1Estimator}
		for i, sampleRate := range sampleRateList {
			samples := sampleData(data, sampleRate)
			for j, name := range estimatorNames {
				estNDV := estimators[j](samples, N)
				qe := qerror(float64(actNDV), estNDV)
				results[j][i] = qe
				fmt.Printf("sample rate: %v, %v NDV:%v, q-error:%v\n", sampleRate, name, estNDV, qe)
			}
		}
		bar := charts.NewBar()
		bar.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: fmt.Sprintf("sample-based NDV estimation, Zipf{s:%v, v:%v, [0, %v], N:%v}", s, v, imax, N)}),
			charts.WithXAxisOpts(opts.XAxis{Name: "sample rate"}),
			charts.WithYAxisOpts(opts.YAxis{Name: "q-error", Type: "log"}),
			charts.WithLegendOpts(opts.Legend{Show: true, Right: "5%", Top: "5%"}),
			charts.WithTooltipOpts(opts.Tooltip{Show: true}))
		bar.SetXAxis(sampleRateList)
		for i, name := range estimatorNames {
			items := make([]opts.BarData, 0, len(sampleRateList))
			for _, qe := range results[i] {
				items = append(items, opts.BarData{Value: qe})
			}
			bar.AddSeries(name, items)
		}
		bars = append(bars, bar)
	}
	page := components.NewPage()
	page.AddCharts(bars...)
	f, err := os.Create("picture/est-ndv.html")
	if err != nil {
		panic(err)
	}
	err = page.Render(io.MultiWriter(f))
	if err != nil {
		panic(err)
	}
}

func testHyperLogLog() {
	s := 1.5
	v := 1.0
	imax := uint64(10000000000)
	N := int64(10000000)
	data := generateZipfData(s, v, imax, N)
	actNDV := exactNDV(data)
	fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], N:%v, NDV:%v\n", s, v, imax, N, actNDV)
	registers := uint32(1024)
	hll, err := NewHyperLogLog(registers)
	if err != nil {
		panic(err)
	}
	for _, x := range data {
		err = hll.InsertUint64(x)
		if err != nil {
			panic(err)
		}
	}
	estNDV := hll.Count()
	qe := qerror(float64(actNDV), float64(estNDV))
	fmt.Printf("HyperLogLog, registers: %v, NDV:%v, q-error:%v\n", registers, estNDV, qe)

}

func main() {
	testHyperLogLog()
}
