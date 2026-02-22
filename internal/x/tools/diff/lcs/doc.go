// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package lcs contains code to find longest-common-subsequences
// (and diffs)
package lcs

/*
Compute longest-common-subsequences of two slices A, B using
algorithms from Myers' paper. A longest-common-subsequence
(LCS from now on) of A and B is a maximal set of lexically increasing
pairs of subscripts (x,y) with A[x]==B[y]. There may be many LCS, but
they all have the same length. An LCS determines a sequence of edits
that changes A into B.

The key concept is the edit graph of A and B.
If A has length N and B has length M, then the edit graph has
vertices v[i][j] for 0 <= i <= N, 0 <= j <= M. There is a
horizontal edge from v[i][j] to v[i+1][j] whenever both are in
the graph, and a vertical edge from v[i][j] to f[i][j+1] similarly.
When A[i] == B[j] there is a diagonal edge from v[i][j] to v[i+1][j+1].

A path between in the graph between (0,0) and (N,M) determines a sequence
of edits converting A into B: each horizontal edge corresponds to removing
an element of A, and each vertical edge corresponds to inserting an
element of B.

Eugene Myers paper is titled
"An O(ND) Difference Algorithm and Its Variations"
and can be found at
http://www.xmailserver.org/diff2.pdf
*/
