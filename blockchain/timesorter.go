// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

// timeSorter implements sort.Interface to allow a slice of timestamps to
// be sorted.
type timeSorter []int64

//Len返回片中的时间戳的数量。
// Len returns the number of timestamps in the slice.  It is part of the
// sort.Interface implementation.
func (s timeSorter) Len() int {
	return len(s)
}
//交换交换通过的索引的时间戳。
// Swap swaps the timestamps at the passed indices.  It is part of the
// sort.Interface implementation.
func (s timeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less返回带有索引i的时间戳是否应该在带有索引j的时间戳之前排序。
// Less returns whether the timstamp with index i should sort before the
// timestamp with index j.  It is part of the sort.Interface implementation.
func (s timeSorter) Less(i, j int) bool {
	return s[i] < s[j]
}
