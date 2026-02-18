package main

import (
	"testing"
)

func TestComputeClues(t *testing.T) {
	tests := []struct {
		guess, answer string
		want          [5]Clue
	}{
		{"hello", "hello", [5]Clue{Correct, Correct, Correct, Correct, Correct}},
		// h=W, e=W, l=W(green l used pos3), l=C, o=Y(o at pos1 of world)
		{"hello", "world", [5]Clue{Wrong, Wrong, Wrong, Correct, Misplaced}},
		// c=W, r=C, a=C, n=Y(n at pos4), e=W
		{"crane", "brain", [5]Clue{Wrong, Correct, Correct, Misplaced, Wrong}},
		// a=C, a=W(a used), b=Y(b at pos1), b=W(b used), c=Y(c at pos2)
		{"aabbc", "abcde", [5]Clue{Correct, Wrong, Misplaced, Wrong, Misplaced}},
		// a=C, a=W(only one a), x=C, x=C, x=C
		{"aaxxx", "axxxx", [5]Clue{Correct, Wrong, Correct, Correct, Correct}},
	}
	for _, tt := range tests {
		got := computeClues(tt.guess, tt.answer)
		if got != tt.want {
			t.Errorf("computeClues(%q, %q) = %v, want %v", tt.guess, tt.answer, got, tt.want)
		}
	}
}

func TestGetValidWords(t *testing.T) {
	words := []string{"crane", "brain", "train", "plain", "drain"}

	// No guesses — all words valid
	got := getValidWords(words, nil)
	if len(got) != 5 {
		t.Errorf("no guesses: got %d words, want 5", len(got))
	}

	// Guess "crane" with correct C at pos 0, rest wrong
	var w [5]byte
	copy(w[:], "crane")
	guesses := []Guess{{Word: w, Clues: [5]Clue{Correct, Wrong, Wrong, Wrong, Wrong}}}
	got = getValidWords(words, guesses)
	for _, word := range got {
		if word == "crane" {
			t.Error("crane should be excluded (it was guessed)")
		}
	}

	// Guess "brain" — B correct at pos 0
	copy(w[:], "brain")
	guesses2 := []Guess{{Word: w, Clues: [5]Clue{Correct, Wrong, Wrong, Wrong, Wrong}}}
	got2 := getValidWords(words, guesses2)
	for _, word := range got2 {
		if word[0] != 'b' {
			t.Errorf("expected words starting with b, got %q", word)
		}
		if word == "brain" {
			t.Error("brain should be excluded (it was guessed)")
		}
	}
}

func TestWordMatchesGuess(t *testing.T) {
	var w [5]byte

	// Green: word must have that letter at that position
	copy(w[:], "crane")
	g := Guess{Word: w, Clues: [5]Clue{Correct, Wrong, Wrong, Wrong, Wrong}}
	if !wordMatchesGuess("cxxxx", g) {
		t.Error("cxxxx should match: c is correct at 0, rest wrong and not in cxxxx")
	}

	// Yellow: letter must be in word but NOT at that position
	copy(w[:], "crane")
	g = Guess{Word: w, Clues: [5]Clue{Misplaced, Wrong, Wrong, Wrong, Wrong}}
	if wordMatchesGuess("cxxxx", g) {
		t.Error("cxxxx should NOT match: c is misplaced at 0 but cxxxx has c at 0")
	}
	if !wordMatchesGuess("xxcxx", g) {
		t.Error("xxcxx should match: c is misplaced at 0 and xxcxx has c elsewhere")
	}

	// Wrong: letter must NOT be in word at all
	copy(w[:], "crane")
	g = Guess{Word: w, Clues: [5]Clue{Wrong, Wrong, Wrong, Wrong, Wrong}}
	if wordMatchesGuess("cxxxx", g) {
		t.Error("cxxxx should NOT match: c is wrong but cxxxx contains c")
	}
	if !wordMatchesGuess("xxxxx", g) {
		t.Error("xxxxx should match: no letters from crane")
	}
}

func TestSolveSmallSet(t *testing.T) {
	candidates := []string{"crane", "crate", "craze", "grace", "trace"}

	// Hard mode: guess pool = candidates
	hard := solve(candidates, candidates, nil)
	if len(hard) == 0 {
		t.Fatal("solve returned no suggestions for hard mode")
	}
	// Best guess should have smallest worst-case
	for _, s := range hard {
		if s.Depth < 0 {
			t.Errorf("negative depth for %s", s.Word)
		}
		if s.Width < 0 {
			t.Errorf("negative width for %s", s.Word)
		}
	}
	// Should be sorted: depth ascending
	for i := 1; i < len(hard); i++ {
		if hard[i].Depth < hard[i-1].Depth {
			t.Errorf("not sorted by depth: %s(%d) before %s(%d)",
				hard[i-1].Word, hard[i-1].Depth, hard[i].Word, hard[i].Depth)
		}
	}
}

func TestSolveSingleCandidate(t *testing.T) {
	candidates := []string{"crane"}
	result := solve(candidates, candidates, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(result))
	}
	if result[0].Word != "crane" {
		t.Errorf("expected crane, got %s", result[0].Word)
	}
	if result[0].Depth != 1 {
		t.Errorf("single candidate depth should be 1, got %d", result[0].Depth)
	}
}

func TestSolveTwoCandidates(t *testing.T) {
	candidates := []string{"crane", "crate"}
	result := solve(candidates, candidates, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(result))
	}
	// With 2 candidates, worst case is 1 guess (guess one, if wrong it's the other)
	for _, s := range result {
		if s.Depth != 1 {
			t.Errorf("two candidates: %s depth should be 1, got %d", s.Word, s.Depth)
		}
	}
}

func TestSolveNormalModeIncludesNonCandidates(t *testing.T) {
	allWords := []string{"crane", "crate", "craze", "xxxxx", "yyyyy", "zzzzz"}
	candidates := []string{"crane", "crate", "craze"}

	normal := solve(allWords, candidates, nil)
	if len(normal) != len(allWords) {
		t.Fatalf("normal mode should have %d suggestions (one per guess word), got %d", len(allWords), len(normal))
	}

	// Check that non-candidate words appear in results
	found := false
	for _, s := range normal {
		if s.Word == "xxxxx" || s.Word == "yyyyy" || s.Word == "zzzzz" {
			found = true
			break
		}
	}
	if !found {
		t.Error("normal mode should include non-candidate words as guesses")
	}
}

func TestComputeCluesDoubleLetters(t *testing.T) {
	// steel vs spell: s=C, t=W, e=C(pos2), e=W(no unused e), l=C
	got := computeClues("steel", "spell")
	want := [5]Clue{Correct, Wrong, Correct, Wrong, Correct}
	if got != want {
		t.Errorf("computeClues(steel, spell) = %v, want %v", got, want)
	}

	// speed vs creep: s=W, p=Y(p at pos4), e=C, e=C, d=W
	got = computeClues("speed", "creep")
	want = [5]Clue{Wrong, Misplaced, Correct, Correct, Wrong}
	if got != want {
		t.Errorf("computeClues(speed, creep) = %v, want %v", got, want)
	}
}
