// The idea is from https://arxiv.org/pdf/2206.05476.pdf.
// Its original implementation is https://github.com/llijiajun/NDV_Estimation_in_distributed_environment.
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
		estimators := []func(samples []uint64, N int64) float64{FirstOrderJackknifeEstimator, SecondOrderJackknifeEstimator, Duj1Estimator}
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
	registers := uint32(1 << 16)
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
	qe := qerror(float64(actNDV), estNDV)
	fmt.Printf("HyperLogLog, registers: %v, NDV:%v, q-error:%v\n", registers, estNDV, qe)
}

func collectSketchFromPartition(partitionData []uint64, sampleRate float64, registers uint32) (*HyperLogLog, *HyperLogLog, float64, error) {
	samples := sampleData(partitionData, sampleRate)
	h := make(map[uint64]uint64, 1000)
	for _, x := range samples {
		h[x] = h[x] + 1
	}
	ndvSketch, err := NewHyperLogLog(registers)
	if err != nil {
		return nil, nil, 0, err
	}
	f1Sketch, err := NewHyperLogLog(registers)
	if err != nil {
		return nil, nil, 0, err
	}
	for x, f := range h {
		err = ndvSketch.InsertUint64(x)
		if err != nil {
			return nil, nil, 0, err
		}
		if f == 1 {
			err = f1Sketch.InsertUint64(x)
			if err != nil {
				return nil, nil, 0, err
			}
		}
	}
	return ndvSketch, f1Sketch, float64(len(samples)), nil
}

func testDistSampleEstimator() {
	s := 1.5
	v := 1.0
	imax := uint64(10000000000)
	N := int64(100000000)
	data := generateZipfData(s, v, imax, N)
	actNDV := exactNDV(data)
	fmt.Printf("zipf dist: s:%v, v:%v, [0, %v], N:%v, NDV:%v\n", s, v, imax, N, actNDV)
	numPartition := int64(10)
	numPerPartition := N / numPartition
	sampleRate := 0.2
	registers := uint32(1 << 16)
	ndvSketches := make([]*HyperLogLog, numPartition)
	f1Sketches := make([]*HyperLogLog, numPartition)
	n := 0.0
	var err error
	for i := int64(0); i < numPartition; i++ {
		partitionData := data[i*numPerPartition : (i+1)*numPerPartition]
		var sampleNum float64
		ndvSketches[i], f1Sketches[i], sampleNum, err = collectSketchFromPartition(partitionData, sampleRate, registers)
		if err != nil {
			panic(err)
		}
		n += sampleNum
	}
	observedNDV, f1, err := EstimateNDVAndF1(ndvSketches, f1Sketches)
	// first-order jackknife estimator
	estimatedNDV := observedNDV + (n-1)/n*f1
	// Chao’s Estimator
	//estimatedNDV := observedNDV + 0.5*f1*f1/(observedNDV-f1)
	qe := qerror(float64(actNDV), estimatedNDV)
	fmt.Printf("dist sample first-order jackknife estimator, partitions:%v, sample rate:%v, registers:%v, NDV:%v, q-error:%v\n", numPartition, sampleRate, registers, estimatedNDV, qe)
}

func main() {
	//benchSampleBasedEstimators()
	//testHyperLogLog()
	testDistSampleEstimator()
}
