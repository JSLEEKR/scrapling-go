package similarity

import (
	"math"
	"testing"
)

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestSequenceMatcherIdentical(t *testing.T) {
	sm := NewSequenceMatcher("hello", "hello")
	if r := sm.Ratio(); r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}

func TestSequenceMatcherEmpty(t *testing.T) {
	sm := NewSequenceMatcher("", "")
	if r := sm.Ratio(); r != 1.0 {
		t.Errorf("expected 1.0 for empty strings, got %f", r)
	}
}

func TestSequenceMatcherOneEmpty(t *testing.T) {
	sm := NewSequenceMatcher("abc", "")
	if r := sm.Ratio(); r != 0.0 {
		t.Errorf("expected 0.0, got %f", r)
	}
}

func TestSequenceMatcherCompleteDiff(t *testing.T) {
	sm := NewSequenceMatcher("abc", "xyz")
	if r := sm.Ratio(); r != 0.0 {
		t.Errorf("expected 0.0, got %f", r)
	}
}

func TestSequenceMatcherPartialMatch(t *testing.T) {
	sm := NewSequenceMatcher("abcd", "bcde")
	r := sm.Ratio()
	// "bcd" matches: 2*3/8 = 0.75
	if !almostEqual(r, 0.75, 0.01) {
		t.Errorf("expected ~0.75, got %f", r)
	}
}

func TestSequenceMatcherKnownResult(t *testing.T) {
	// "abcde" vs "acedb": matching "a" + "e" = 2 chars, ratio = 2*2/10 = 0.4
	sm := NewSequenceMatcher("abcde", "acedb")
	r := sm.Ratio()
	if !almostEqual(r, 0.4, 0.05) {
		t.Errorf("expected ~0.4, got %f", r)
	}
}

func TestSequenceMatcherHighSimilarity(t *testing.T) {
	// "private", "privat" -> "privat" matches, ratio = 2*6/13 ≈ 0.923
	sm := NewSequenceMatcher("private", "privat")
	r := sm.Ratio()
	if !almostEqual(r, 0.923, 0.01) {
		t.Errorf("expected ~0.923, got %f", r)
	}
}

func TestSequenceMatcherLongStrings(t *testing.T) {
	a := "the quick brown fox jumps over the lazy dog"
	b := "the quick brown fox jumped over a lazy cat"
	sm := NewSequenceMatcher(a, b)
	r := sm.Ratio()
	if r < 0.7 || r > 1.0 {
		t.Errorf("expected high similarity, got %f", r)
	}
}

func TestStringRatio(t *testing.T) {
	r := StringRatio("hello", "hello")
	if r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}

func TestStringRatioPartial(t *testing.T) {
	r := StringRatio("abc", "abd")
	// "ab" matches: 2*2/6 ≈ 0.667
	if !almostEqual(r, 0.667, 0.01) {
		t.Errorf("expected ~0.667, got %f", r)
	}
}

func TestSetRatioIdentical(t *testing.T) {
	r := SetRatio([]string{"a", "b", "c"}, []string{"a", "b", "c"})
	if r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}

func TestSetRatioDisjoint(t *testing.T) {
	r := SetRatio([]string{"a", "b"}, []string{"c", "d"})
	if r != 0.0 {
		t.Errorf("expected 0.0, got %f", r)
	}
}

func TestSetRatioPartial(t *testing.T) {
	r := SetRatio([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	// intersection=2, union=4 → 0.5
	if !almostEqual(r, 0.5, 0.01) {
		t.Errorf("expected 0.5, got %f", r)
	}
}

func TestSetRatioBothEmpty(t *testing.T) {
	r := SetRatio([]string{}, []string{})
	if r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}

func TestSetRatioOneEmpty(t *testing.T) {
	r := SetRatio([]string{"a"}, []string{})
	if r != 0.0 {
		t.Errorf("expected 0.0, got %f", r)
	}
}

func TestGetMatchingBlocks(t *testing.T) {
	sm := NewSequenceMatcher("abxcd", "abcd")
	blocks := sm.GetMatchingBlocks()
	// Should find "ab" and "cd" matches + sentinel
	if len(blocks) < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", len(blocks))
	}
	last := blocks[len(blocks)-1]
	if last.Size != 0 {
		t.Error("last block should be sentinel with size 0")
	}
}

func TestGetOpcodes(t *testing.T) {
	sm := NewSequenceMatcher("abcd", "abef")
	ops := sm.GetOpcodes()
	if len(ops) == 0 {
		t.Fatal("expected opcodes")
	}
	// First op should be 'e' for "ab"
	if ops[0].Tag != 'e' {
		t.Errorf("expected 'e' tag, got %c", ops[0].Tag)
	}
}

func TestQuickRatio(t *testing.T) {
	sm := NewSequenceMatcher("abcd", "bcde")
	qr := sm.QuickRatio()
	r := sm.Ratio()
	if qr < r {
		t.Errorf("QuickRatio %f should be >= Ratio %f", qr, r)
	}
}

func TestSetSeqs(t *testing.T) {
	sm := NewSequenceMatcher("abc", "abc")
	if sm.Ratio() != 1.0 {
		t.Error("expected 1.0")
	}
	sm.SetSeqs("abc", "xyz")
	if sm.Ratio() != 0.0 {
		t.Error("expected 0.0 after SetSeqs")
	}
}

func TestSetA(t *testing.T) {
	sm := NewSequenceMatcher("abc", "abc")
	sm.SetA("xyz")
	if sm.Ratio() == 1.0 {
		t.Error("should not be 1.0 after changing A")
	}
}

func TestSetB(t *testing.T) {
	sm := NewSequenceMatcher("abc", "abc")
	sm.SetB("xyz")
	if sm.Ratio() == 1.0 {
		t.Error("should not be 1.0 after changing B")
	}
}

func TestMatchingBlocksCaching(t *testing.T) {
	sm := NewSequenceMatcher("hello world", "hello earth")
	b1 := sm.GetMatchingBlocks()
	b2 := sm.GetMatchingBlocks()
	if len(b1) != len(b2) {
		t.Error("cached result should match")
	}
}

func TestSequenceMatcherReversed(t *testing.T) {
	r1 := StringRatio("abc", "bcd")
	r2 := StringRatio("bcd", "abc")
	if !almostEqual(r1, r2, 0.001) {
		t.Errorf("ratio should be symmetric: %f vs %f", r1, r2)
	}
}

func TestSequenceMatcherSingleChar(t *testing.T) {
	r := StringRatio("a", "a")
	if r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}

func TestSequenceMatcherRepeatChars(t *testing.T) {
	r := StringRatio("aaa", "aaa")
	if r != 1.0 {
		t.Errorf("expected 1.0, got %f", r)
	}
}
