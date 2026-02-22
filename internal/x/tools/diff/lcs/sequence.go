// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lcs

// This file defines the abstract sequence over which the LCS algorithm operates.

// sequences abstracts a pair of sequences, A and B.
type sequences interface {
	lengths() (int, int)
	commonPrefixLen(ai, aj, bi, bj int) int
	commonSuffixLen(ai, aj, bi, bj int) int
}

// The explicit capacity in s[i:j:j] leads to more efficient code.

type bytesSeqs struct{ a, b []byte }

func (s bytesSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s bytesSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenSlice(s.a[ai:aj:aj], s.b[bi:bj:bj])
}
func (s bytesSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenSlice(s.a[ai:aj:aj], s.b[bi:bj:bj])
}

type runesSeqs struct{ a, b []rune }

func (s runesSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s runesSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenSlice(s.a[ai:aj:aj], s.b[bi:bj:bj])
}
func (s runesSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenSlice(s.a[ai:aj:aj], s.b[bi:bj:bj])
}

type linesSeqs struct{ a, b []string }

func (s linesSeqs) lengths() (int, int) { return len(s.a), len(s.b) }
func (s linesSeqs) commonPrefixLen(ai, aj, bi, bj int) int {
	return commonPrefixLenSlice(s.a[ai:aj], s.b[bi:bj])
}
func (s linesSeqs) commonSuffixLen(ai, aj, bi, bj int) int {
	return commonSuffixLenSlice(s.a[ai:aj], s.b[bi:bj])
}

// commonPrefixLenSlice returns the length of the common prefix of a and b.
func commonPrefixLenSlice[T comparable](a, b []T) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

// commonSuffixLenSlice returns the length of the common suffix of a and b.
func commonSuffixLenSlice[T comparable](a, b []T) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
