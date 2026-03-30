// Package similarity provides a SequenceMatcher implementation equivalent to
// Python's difflib.SequenceMatcher for computing string similarity ratios.
package similarity

// SequenceMatcher compares two sequences and finds the longest common subsequences.
// It is a Go port of Python's difflib.SequenceMatcher.
type SequenceMatcher struct {
	a, b     []rune
	aStr     string
	bStr     string
	bIndex   map[rune][]int
	opcodes  []OpCode
	mbCache  []Match
}

// OpCode represents a single edit operation.
type OpCode struct {
	Tag  byte // 'e' equal, 'r' replace, 'i' insert, 'd' delete
	I1   int
	I2   int
	J1   int
	J2   int
}

// Match represents a matching block between two sequences.
type Match struct {
	A    int // position in sequence a
	B    int // position in sequence b
	Size int // length of match
}

// NewSequenceMatcher creates a new SequenceMatcher for the given sequences.
func NewSequenceMatcher(a, b string) *SequenceMatcher {
	sm := &SequenceMatcher{
		a:    []rune(a),
		b:    []rune(b),
		aStr: a,
		bStr: b,
	}
	sm.chainB()
	return sm
}

// chainB indexes every rune position in sequence b for fast lookup.
func (sm *SequenceMatcher) chainB() {
	sm.bIndex = make(map[rune][]int, len(sm.b))
	for i, r := range sm.b {
		sm.bIndex[r] = append(sm.bIndex[r], i)
	}
}

// SetSeqs replaces both sequences and resets caches.
func (sm *SequenceMatcher) SetSeqs(a, b string) {
	sm.a = []rune(a)
	sm.b = []rune(b)
	sm.aStr = a
	sm.bStr = b
	sm.opcodes = nil
	sm.mbCache = nil
	sm.chainB()
}

// SetA replaces sequence a.
func (sm *SequenceMatcher) SetA(a string) {
	sm.a = []rune(a)
	sm.aStr = a
	sm.opcodes = nil
	sm.mbCache = nil
}

// SetB replaces sequence b and reindexes.
func (sm *SequenceMatcher) SetB(b string) {
	sm.b = []rune(b)
	sm.bStr = b
	sm.opcodes = nil
	sm.mbCache = nil
	sm.chainB()
}

// findLongestMatch finds the longest matching block in a[alo:ahi] and b[blo:bhi].
func (sm *SequenceMatcher) findLongestMatch(alo, ahi, blo, bhi int) Match {
	bestI, bestJ, bestSize := alo, blo, 0

	// j2len[j] tracks the length of the longest match ending at b[j]
	j2len := make(map[int]int)

	for i := alo; i < ahi; i++ {
		newJ2len := make(map[int]int)
		indices := sm.bIndex[sm.a[i]]
		for _, j := range indices {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newJ2len[j] = k
			if k > bestSize {
				bestI = i - k + 1
				bestJ = j - k + 1
				bestSize = k
			}
		}
		j2len = newJ2len
	}

	// Extend the match as far as possible (handles junk characters edge case)
	for bestI > alo && bestJ > blo && sm.a[bestI-1] == sm.b[bestJ-1] {
		bestI--
		bestJ--
		bestSize++
	}
	for bestI+bestSize < ahi && bestJ+bestSize < bhi && sm.a[bestI+bestSize] == sm.b[bestJ+bestSize] {
		bestSize++
	}

	return Match{A: bestI, B: bestJ, Size: bestSize}
}

// GetMatchingBlocks returns a list of matching blocks. The last block is a
// sentinel (len(a), len(b), 0).
func (sm *SequenceMatcher) GetMatchingBlocks() []Match {
	if sm.mbCache != nil {
		return sm.mbCache
	}

	la, lb := len(sm.a), len(sm.b)
	var matches []Match

	type span struct {
		alo, ahi, blo, bhi int
	}
	queue := []span{{0, la, 0, lb}}

	for len(queue) > 0 {
		s := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		m := sm.findLongestMatch(s.alo, s.ahi, s.blo, s.bhi)
		if m.Size > 0 {
			matches = append(matches, m)
			if s.alo < m.A && s.blo < m.B {
				queue = append(queue, span{s.alo, m.A, s.blo, m.B})
			}
			if m.A+m.Size < s.ahi && m.B+m.Size < s.bhi {
				queue = append(queue, span{m.A + m.Size, s.ahi, m.B + m.Size, s.bhi})
			}
		}
	}

	// Sort by a position
	sortMatches(matches)

	// Collapse adjacent equal blocks
	var result []Match
	i1, j1, k1 := 0, 0, 0
	for _, m := range matches {
		if i1+k1 == m.A && j1+k1 == m.B {
			k1 += m.Size
		} else {
			if k1 > 0 {
				result = append(result, Match{A: i1, B: j1, Size: k1})
			}
			i1, j1, k1 = m.A, m.B, m.Size
		}
	}
	if k1 > 0 {
		result = append(result, Match{A: i1, B: j1, Size: k1})
	}
	result = append(result, Match{A: la, B: lb, Size: 0})
	sm.mbCache = result
	return result
}

// sortMatches sorts matches by A position using insertion sort (typically small).
func sortMatches(matches []Match) {
	for i := 1; i < len(matches); i++ {
		key := matches[i]
		j := i - 1
		for j >= 0 && (matches[j].A > key.A || (matches[j].A == key.A && matches[j].B > key.B)) {
			matches[j+1] = matches[j]
			j--
		}
		matches[j+1] = key
	}
}

// GetOpcodes returns a list of 5-tuples describing how to turn a into b.
func (sm *SequenceMatcher) GetOpcodes() []OpCode {
	if sm.opcodes != nil {
		return sm.opcodes
	}

	i, j := 0, 0
	var opcodes []OpCode
	for _, m := range sm.GetMatchingBlocks() {
		tag := byte(0)
		if i < m.A && j < m.B {
			tag = 'r'
		} else if i < m.A {
			tag = 'd'
		} else if j < m.B {
			tag = 'i'
		}
		if tag != 0 {
			opcodes = append(opcodes, OpCode{Tag: tag, I1: i, I2: m.A, J1: j, J2: m.B})
		}
		i, j = m.A+m.Size, m.B+m.Size
		if m.Size > 0 {
			opcodes = append(opcodes, OpCode{Tag: 'e', I1: m.A, I2: i, J1: m.B, J2: j})
		}
	}
	sm.opcodes = opcodes
	return opcodes
}

// Ratio returns a float in [0, 1] measuring similarity.
// 2.0 * M / T where M = matching characters, T = total characters in both.
func (sm *SequenceMatcher) Ratio() float64 {
	totalLen := len(sm.a) + len(sm.b)
	if totalLen == 0 {
		return 1.0
	}
	matches := 0
	for _, m := range sm.GetMatchingBlocks() {
		matches += m.Size
	}
	return 2.0 * float64(matches) / float64(totalLen)
}

// QuickRatio returns an upper bound on Ratio() relatively quickly.
func (sm *SequenceMatcher) QuickRatio() float64 {
	totalLen := len(sm.a) + len(sm.b)
	if totalLen == 0 {
		return 1.0
	}

	// Count rune frequencies
	freqA := make(map[rune]int)
	freqB := make(map[rune]int)
	for _, r := range sm.a {
		freqA[r]++
	}
	for _, r := range sm.b {
		freqB[r]++
	}

	matches := 0
	for ch, countA := range freqA {
		if countB, ok := freqB[ch]; ok {
			if countA < countB {
				matches += countA
			} else {
				matches += countB
			}
		}
	}
	return 2.0 * float64(matches) / float64(totalLen)
}

// StringRatio is a convenience function that returns the similarity ratio
// between two strings.
func StringRatio(a, b string) float64 {
	sm := NewSequenceMatcher(a, b)
	return sm.Ratio()
}

// SetRatio computes the similarity ratio between two string sets
// using intersection/union (Jaccard similarity).
func SetRatio(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	setA := make(map[string]bool, len(a))
	for _, s := range a {
		setA[s] = true
	}
	setB := make(map[string]bool, len(b))
	for _, s := range b {
		setB[s] = true
	}

	intersection := 0
	for s := range setA {
		if setB[s] {
			intersection++
		}
	}

	union := len(setA)
	for s := range setB {
		if !setA[s] {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}
	return float64(intersection) / float64(union)
}
