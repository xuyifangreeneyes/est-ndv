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

// estimateF1 returns f1. f1 means the number of values which exist exactly once. Assume there are k partitions.
// f1 = v1 + v2 + ... + vk, where vi is the number of values which exist in partition i exactly once and don't exist in other partitions.
// vi = NDV(Xi U Yi) - NDV(Yi), where Xi is the set containing values Yi is which exist in partition i exactly once,
// and Yi is the set containing all partitions except partition i.
func (e *DistSampleEstimator) estimateF1() (uint64, error) {
	var f1 float64

}

func (e *DistSampleEstimator) estimateNDV() (uint64, error) {
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

func EstimateNDVAndF1(ndvSketches, f1Sketches []*HyperLogLog) (uint64, uint64, error) {
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
	f1 := e.estimateF1()
	ndv, err := e.estimateNDV()
	if err != nil {
		return 0, 0, err
	}
	return ndv, f1, nil
}
