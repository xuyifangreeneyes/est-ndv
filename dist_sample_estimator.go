package main

import "fmt"

type DistSampleEstimator struct {
	ndvSketches []*HyperLogLog
	f1Sketches  []*HyperLogLog
	segmentTree []*HyperLogLog
}

func (e *DistSampleEstimator) buildSegmentTree(s, t, p int) (err error) {
	if s == t {
		e.segmentTree[p] = e.ndvSketches[s].Clone()
		return
	}
	m := s + (t-s)/2
	err = e.buildSegmentTree(s, m, 2*p)
	if err != nil {
		return err
	}
	err = e.buildSegmentTree(m+1, t, 2*p+1)
	if err != nil {
		return err
	}
	e.segmentTree[p], err = NewHyperLogLog(e.ndvSketches[0].GetRegisterNum())
	if err != nil {
		return err
	}
	err = e.segmentTree[p].Merge(e.segmentTree[2*p])
	if err != nil {
		return err
	}
	err = e.segmentTree[p].Merge(e.segmentTree[2*p+1])
	if err != nil {
		return err
	}
	return nil
}

func (e *DistSampleEstimator) querySegmentTree(l, r, s, t, p int) (*HyperLogLog, error) {
	if l <= s && t <= r {
		return e.segmentTree[p].Clone(), nil
	}
	m := s + (t-s)/2
	res, err := NewHyperLogLog(e.ndvSketches[0].GetRegisterNum())
	if err != nil {
		return nil, err
	}
	if l <= m {
		left, err := e.querySegmentTree(l, r, s, m, 2*p)
		if err != nil {
			return nil, err
		}
		err = res.Merge(left)
		if err != nil {
			return nil, err
		}
	}
	if r > m {
		right, err := e.querySegmentTree(l, r, m+1, t, 2*p+1)
		if err != nil {
			return nil, err
		}
		err = res.Merge(right)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// getSketchForInterval returns sketch for [l, r] partitions.
func (e *DistSampleEstimator) getSketchForInterval(l, r int) (*HyperLogLog, error) {
	num := len(e.ndvSketches)
	return e.querySegmentTree(l, r, 0, num-1, 1)
}

// getSketchForOtherPartitions returns sketch for the set containing all partitions except partition i.
func (e *DistSampleEstimator) getSketchForOtherPartitions(i int) (*HyperLogLog, error) {
	num := len(e.ndvSketches)
	if i == 0 {
		return e.getSketchForInterval(1, num-1)
	}
	if i == num-1 {
		return e.getSketchForInterval(0, num-2)
	}
	s1, err := e.getSketchForInterval(0, i-1)
	if err != nil {
		return nil, err
	}
	s2, err := e.getSketchForInterval(i+1, num-1)
	if err != nil {
		return nil, err
	}
	err = s1.Merge(s2)
	if err != nil {
		return nil, err
	}
	return s1, nil
}

// estimateF1 returns f1. f1 means the number of values which exist exactly once. Assume there are k partitions.
// f1 = v1 + v2 + ... + vk, where vi is the number of values which exist in partition i exactly once and don't exist in
// other partitions.
// vi = NDV(Xi U Yi) - NDV(Yi), where Xi is the set containing values which exist in partition i exactly once, and Yi is
// the set containing all partitions except partition i.
func (e *DistSampleEstimator) estimateF1() (float64, error) {
	var f1 float64
	num := len(e.ndvSketches)
	for i := 0; i < num; i++ {
		ySketch, err := e.getSketchForOtherPartitions(i)
		if err != nil {
			return 0, err
		}
		f1 -= ySketch.Count()
		xySketch := e.f1Sketches[i].Clone()
		err = xySketch.Merge(ySketch)
		if err != nil {
			return 0, err
		}
		f1 += xySketch.Count()
	}
	return f1, nil
}

func (e *DistSampleEstimator) estimateNDV() (float64, error) {
	res, err := NewHyperLogLog(e.ndvSketches[0].GetRegisterNum())
	if err != nil {
		return 0, err
	}
	for _, sketch := range e.ndvSketches {
		err = res.Merge(sketch)
		if err != nil {
			return 0, err
		}
	}
	return res.Count(), nil
}

func EstimateNDVAndF1(ndvSketches, f1Sketches []*HyperLogLog) (float64, float64, error) {
	if len(ndvSketches) != len(f1Sketches) {
		return 0, 0, fmt.Errorf("number of ndvSketches %v not equal to number of f1Sketches %v", len(ndvSketches), len(f1Sketches))
	}
	if len(ndvSketches) == 0 {
		return 0, 0, fmt.Errorf("no sketch")
	}
	num := len(ndvSketches)
	e := &DistSampleEstimator{
		ndvSketches: ndvSketches,
		f1Sketches:  f1Sketches,
		segmentTree: make([]*HyperLogLog, 4*num),
	}
	err := e.buildSegmentTree(0, num-1, 1)
	if err != nil {
		return 0, 0, err
	}
	f1, err := e.estimateF1()
	if err != nil {
		return 0, 0, err
	}
	ndv, err := e.estimateNDV()
	if err != nil {
		return 0, 0, err
	}
	return ndv, f1, nil
}
