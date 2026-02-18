package main

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2/data/binding"
)

const (
	Rows = 6
	Cols = 5
)

// GameState holds all bindable game data for the Wordle solver GUI.
// Words[i] is the 5-letter guess for row i.
// Clues[i][j] is the clue for row i, column j.
// Solve() filters candidates and populates HardSuggestions and NormalSuggestions.
type GameState struct {
	Words   [Rows]binding.String       // lowercase 5-letter word per row, or ""
	Clues   [Rows][Cols]binding.Int    // 0=Wrong, 1=Misplaced, 2=Correct
	Letters [Rows][Cols]binding.String // uppercase letter per cell for display

	// Cursor position (-1,-1 means no selection)
	CursorRow binding.Int
	CursorCol binding.Int

	Candidates        binding.StringList // remaining possible answers
	CandidateCount    binding.Int        // len(Candidates) for easy label binding
	HardSuggestions   binding.StringList // pre-formatted suggestion strings (hard mode)
	NormalSuggestions binding.StringList // pre-formatted suggestion strings (normal mode)
	Solving           binding.Bool       // true while Solve() is running
	Progress          binding.Float      // 0.0 to 1.0 solve progress
	StatusMessage     binding.String     // info/warning/error message for the UI

	allWords []string
	mu       sync.Mutex
}

// NewGameState creates a fully initialized GameState ready for binding.
func NewGameState() *GameState {
	gs := &GameState{
		Candidates:        binding.NewStringList(),
		CandidateCount:    binding.NewInt(),
		HardSuggestions:   binding.NewStringList(),
		NormalSuggestions: binding.NewStringList(),
		Solving:           binding.NewBool(),
		Progress:          binding.NewFloat(),
		StatusMessage:     binding.NewString(),
		CursorRow:         binding.NewInt(),
		CursorCol:         binding.NewInt(),
		allWords:          loadWords(),
	}
	gs.CursorRow.Set(-1)
	gs.CursorCol.Set(-1)
	for r := 0; r < Rows; r++ {
		gs.Words[r] = binding.NewString()
		for c := 0; c < Cols; c++ {
			gs.Clues[r][c] = binding.NewInt()
			gs.Letters[r][c] = binding.NewString()
		}
	}
	gs.Candidates.Set(gs.allWords)
	gs.CandidateCount.Set(len(gs.allWords))
	return gs
}

// CycleClue increments Clues[row][col] through Gray→Yellow→Green→Gray.
func (gs *GameState) CycleClue(row, col int) {
	val, _ := gs.Clues[row][col].Get()
	gs.Clues[row][col].Set((val + 1) % 3)
}

// Reset clears the entire board.
func (gs *GameState) Reset() {
	gs.Deselect()
	for r := 0; r < Rows; r++ {
		gs.Words[r].Set("")
		for c := 0; c < Cols; c++ {
			gs.Letters[r][c].Set("")
			gs.Clues[r][c].Set(int(Wrong))
		}
	}
	gs.Candidates.Set(nil)
	gs.CandidateCount.Set(0)
	gs.HardSuggestions.Set(nil)
	gs.NormalSuggestions.Set(nil)
	gs.StatusMessage.Set("")
}

// SelectCell sets the cursor to (row, col).
func (gs *GameState) SelectCell(row, col int) {
	gs.CursorRow.Set(row)
	gs.CursorCol.Set(col)
}

// Deselect clears the cursor.
func (gs *GameState) Deselect() {
	gs.CursorRow.Set(-1)
	gs.CursorCol.Set(-1)
}

// IsCursor returns true if (row, col) is the current cursor position.
func (gs *GameState) IsCursor(row, col int) bool {
	r, _ := gs.CursorRow.Get()
	c, _ := gs.CursorCol.Get()
	return r == row && c == col
}

// TypeLetter sets a letter at the cursor and advances to the next cell.
func (gs *GameState) TypeLetter(ch rune) {
	r, _ := gs.CursorRow.Get()
	c, _ := gs.CursorCol.Get()
	if r < 0 || r >= Rows || c < 0 || c >= Cols {
		return
	}
	letter := strings.ToUpper(string(ch))
	gs.Letters[r][c].Set(letter)
	gs.syncWordFromLetters(r)

	// Advance cursor
	if c+1 < Cols {
		gs.CursorCol.Set(c + 1)
	} else if r+1 < Rows {
		// Move to start of next row
		gs.CursorRow.Set(r + 1)
		gs.CursorCol.Set(0)
	} else {
		gs.Deselect()
	}
}

// Backspace clears the current cell (or previous if current is empty) and moves cursor back.
func (gs *GameState) Backspace() {
	r, _ := gs.CursorRow.Get()
	c, _ := gs.CursorCol.Get()
	if r < 0 || r >= Rows {
		return
	}

	// If current cell has a letter, clear it
	cur, _ := gs.Letters[r][c].Get()
	if cur != "" {
		gs.Letters[r][c].Set("")
		gs.Clues[r][c].Set(int(Wrong))
		gs.syncWordFromLetters(r)
		return
	}

	// Otherwise move back and clear that cell
	if c > 0 {
		c--
		gs.CursorCol.Set(c)
		gs.Letters[r][c].Set("")
		gs.Clues[r][c].Set(int(Wrong))
		gs.syncWordFromLetters(r)
	} else if r > 0 {
		// Move to end of previous row
		r--
		gs.CursorRow.Set(r)
		c = Cols - 1
		gs.CursorCol.Set(c)
		gs.Letters[r][c].Set("")
		gs.Clues[r][c].Set(int(Wrong))
		gs.syncWordFromLetters(r)
	}
}

// syncWordFromLetters rebuilds Words[row] from Letters[row][0..4].
func (gs *GameState) syncWordFromLetters(row int) {
	var word string
	for c := 0; c < Cols; c++ {
		l, _ := gs.Letters[row][c].Get()
		if l == "" {
			word += " "
		} else {
			word += strings.ToLower(l)
		}
	}
	trimmed := strings.TrimSpace(word)
	if len(trimmed) == 5 && !strings.Contains(trimmed, " ") {
		gs.Words[row].Set(trimmed)
	} else {
		gs.Words[row].Set("")
	}
}

// syncLettersFromWord populates Letters[row] from Words[row] (used by InsertWord).
func (gs *GameState) syncLettersFromWord(row int) {
	w, _ := gs.Words[row].Get()
	for c := 0; c < Cols; c++ {
		if c < len(w) {
			gs.Letters[row][c].Set(strings.ToUpper(string(w[c])))
		} else {
			gs.Letters[row][c].Set("")
		}
	}
}

// InsertWord extracts the word from a formatted suggestion string
// and sets it on the first empty row.
func (gs *GameState) InsertWord(formatted string) {
	word := strings.ToLower(strings.Fields(formatted)[0])
	for r := 0; r < Rows; r++ {
		w, _ := gs.Words[r].Get()
		if strings.TrimSpace(w) == "" {
			gs.Words[r].Set(word)
			gs.syncLettersFromWord(r)
			return
		}
	}
}

// formatSuggestions converts []Suggestion to []string for binding.StringList.
func formatSuggestions(suggestions []Suggestion, candidateCount int) []string {
	result := make([]string, len(suggestions))
	for i, s := range suggestions {
		if candidateCount == 1 {
			// Only one candidate — it's the answer
			result[i] = fmt.Sprintf("%s ✓", strings.ToUpper(s.Word))
		} else {
			result[i] = fmt.Sprintf("%s  ⌊%d⌋  %d left",
				strings.ToUpper(s.Word), s.Depth, s.Width)
		}
	}
	return result
}

// Solve reads the current board, filters candidates, scores guesses,
// and updates the bound output lists. Safe to call from a goroutine.
func (gs *GameState) Solve() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Solving.Set(true)
	defer gs.Solving.Set(false)
	gs.StatusMessage.Set("")

	// Read and validate guesses from bound data
	var guesses []Guess
	var warnings []string
	allRowsGray := true // track if ALL filled rows have all-gray clues
	hasAnyRow := false
	for r := 0; r < Rows; r++ {
		word, _ := gs.Words[r].Get()
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			// Check for partially filled rows
			hasLetter := false
			for c := 0; c < Cols; c++ {
				l, _ := gs.Letters[r][c].Get()
				if l != "" {
					hasLetter = true
					break
				}
			}
			if hasLetter {
				warnings = append(warnings, fmt.Sprintf("Row %d: incomplete word (skipped)", r+1))
			}
			continue
		}
		if len(word) != 5 {
			continue
		}
		valid := true
		for _, ch := range word {
			if ch < 'a' || ch > 'z' {
				valid = false
				break
			}
		}
		if !valid {
			warnings = append(warnings, fmt.Sprintf("Row %d: invalid characters (skipped)", r+1))
			continue
		}

		var g Guess
		copy(g.Word[:], word)

		allWrong := true
		allCorrect := true
		for c := 0; c < Cols; c++ {
			val, _ := gs.Clues[r][c].Get()
			g.Clues[c] = Clue(val)
			if Clue(val) != Wrong {
				allWrong = false
			}
			if Clue(val) != Correct {
				allCorrect = false
			}
		}

		if allCorrect {
			// Puzzle is solved — no need to continue
			gs.StatusMessage.Set(fmt.Sprintf("Solved! The answer is %s", strings.ToUpper(word)))
			gs.Candidates.Set([]string{word})
			gs.CandidateCount.Set(1)
			gs.HardSuggestions.Set(formatSuggestions([]Suggestion{{Word: word, Depth: 0, Width: 1}}, 1))
			gs.NormalSuggestions.Set(formatSuggestions([]Suggestion{{Word: word, Depth: 0, Width: 1}}, 1))
			return
		}

		if !allWrong {
			allRowsGray = false
		}
		hasAnyRow = true
		guesses = append(guesses, g)
	}

	if hasAnyRow && allRowsGray {
		warnings = append(warnings, "All rows are gray — did you set the clue colors?")
	}

	if len(guesses) == 0 {
		gs.StatusMessage.Set("No guesses entered — showing best openers")
	}

	// Filter
	cands := getValidWords(gs.allWords, guesses)
	gs.Candidates.Set(cands)
	gs.CandidateCount.Set(len(cands))

	if len(cands) == 0 {
		msg := "No possible answers! Check your clue colors."
		if len(warnings) > 0 {
			msg += "\n" + strings.Join(warnings, "\n")
		}
		gs.StatusMessage.Set(msg)
		gs.HardSuggestions.Set(nil)
		gs.NormalSuggestions.Set(nil)
		return
	}

	// Build status
	status := fmt.Sprintf("%d candidates remaining", len(cands))
	if len(warnings) > 0 {
		status += " ⚠ " + strings.Join(warnings, "; ")
	}
	gs.StatusMessage.Set(status)

	// Cap results
	capSuggestions := func(s []Suggestion) []Suggestion {
		if len(s) > 25 {
			return s[:25]
		}
		return s
	}

	// Clear previous results to avoid stale bindings in list widgets
	gs.HardSuggestions.Set([]string{})
	gs.NormalSuggestions.Set([]string{})

	// Progress: hard mode = 0..0.5, normal mode = 0.5..1.0
	gs.Progress.Set(0)

	// Hard mode: can only guess remaining candidates
	hard := solve(cands, cands, func(p float64) {
		gs.Progress.Set(p * 0.5)
	})
	gs.HardSuggestions.Set(formatSuggestions(capSuggestions(hard), len(cands)))

	// Normal mode: can guess any word
	normal := solve(gs.allWords, cands, func(p float64) {
		gs.Progress.Set(0.5 + p*0.5)
	})
	gs.NormalSuggestions.Set(formatSuggestions(capSuggestions(normal), len(cands)))
}
