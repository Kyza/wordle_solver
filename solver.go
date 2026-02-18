package main

import (
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// Clue represents Wordle feedback for one letter position.
type Clue int

const (
	Wrong     Clue = 0
	Misplaced Clue = 1
	Correct   Clue = 2
)

// Guess is a word paired with its per-letter feedback.
type Guess struct {
	Word  [5]byte
	Clues [5]Clue
}

// Suggestion is a scored guess showing its worst-case solve path.
type Suggestion struct {
	Word        string // the guess word
	WorstAnswer string // answer that produces the hardest partition
	Depth       int    // worst-case number of guesses to solve
	Width       int    // size of the largest remaining partition
}

var allCorrect = [5]Clue{Correct, Correct, Correct, Correct, Correct}

// worstCaseDepth returns the minimum number of guesses needed to guarantee
// identifying any answer in candidates, searching up to maxDepth levels deep.
func worstCaseDepth(candidates []string, maxDepth int) int {
	if len(candidates) <= 1 {
		return 0
	}
	if len(candidates) == 2 {
		return 1 // guess one, if wrong it's the other
	}
	if maxDepth <= 0 {
		return len(candidates) // heuristic fallback
	}

	best := len(candidates) // worst possible

	for _, guess := range candidates {
		// Partition candidates by feedback
		buckets := map[[5]Clue][]string{}
		for _, answer := range candidates {
			fb := computeClues(guess, answer)
			buckets[fb] = append(buckets[fb], answer)
		}

		// Worst bucket for this guess (excluding the solved case)
		guessWorst := 0
		for fb, bucket := range buckets {
			if fb == allCorrect {
				continue
			}
			d := 1 + worstCaseDepth(bucket, maxDepth-1)
			if d > guessWorst {
				guessWorst = d
			}
		}

		if guessWorst < best {
			best = guessWorst
		}
	}
	return best
}

// wordMatchesGuess returns true if word is consistent with the observed feedback.
func wordMatchesGuess(word string, guess Guess) bool {
	return computeClues(string(guess.Word[:]), word) == guess.Clues
}

func computeClues(guess string, answer string) [5]Clue {
	var clues [5]Clue
	var used [5]bool

	// Green pass: exact matches
	for i := 0; i < 5; i++ {
		if guess[i] == answer[i] {
			clues[i] = Correct
			used[i] = true
		}
	}

	// Yellow pass: right letter, wrong position
	for i := 0; i < 5; i++ {
		if clues[i] == Correct {
			continue
		}
		for j := 0; j < 5; j++ {
			if !used[j] && guess[i] == answer[j] {
				clues[i] = Misplaced
				used[j] = true
				break
			}
		}
		// If not found, stays Wrong (0)
	}

	return clues
}

// solve scores each word in guessPool against the remaining candidates.
// For hard mode: guessPool = candidates. For normal mode: guessPool = allWords.
// onProgress is called periodically with a value from 0.0 to 1.0 (may be nil).
func solve(guessPool []string, candidates []string, onProgress func(float64)) []Suggestion {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return []Suggestion{{
			Word:        candidates[0],
			WorstAnswer: candidates[0],
			Depth:       1,
			Width:       1,
		}}
	}

	// Adaptive recursion depth: deeper search for smaller candidate sets
	maxDepth := 1
	if len(candidates) <= 50 {
		maxDepth = 3
	} else if len(candidates) <= 300 {
		maxDepth = 2
	}

	suggestions := make([]Suggestion, len(guessPool))
	total := int64(len(guessPool))
	var done atomic.Int64

	scoreRange := func(start, end int) {
		for i := start; i < end; i++ {
			guess := guessPool[i]
			// Partition candidates by feedback
			buckets := map[[5]Clue][]string{}
			for _, answer := range candidates {
				fb := computeClues(guess, answer)
				buckets[fb] = append(buckets[fb], answer)
			}

			// Find the worst bucket: deepest path, then most candidates as tiebreak
			worstAnswer := ""
			worstDepth := 0
			worstWidth := 0

			for fb, bucket := range buckets {
				if fb == allCorrect {
					continue
				}
				d := 1 + worstCaseDepth(bucket, maxDepth-1)
				if d > worstDepth || (d == worstDepth && len(bucket) > worstWidth) {
					worstDepth = d
					worstWidth = len(bucket)
					worstAnswer = bucket[0]
				}
			}

			suggestions[i] = Suggestion{
				Word:        guess,
				WorstAnswer: worstAnswer,
				Depth:       worstDepth,
				Width:       worstWidth,
			}

			n := done.Add(1)
			if onProgress != nil && (n%50 == 0 || n == total) {
				onProgress(float64(n) / float64(total))
			}
		}
	}

	// Parallelize across CPUs
	numWorkers := runtime.NumCPU()
	if numWorkers > len(guessPool) {
		numWorkers = len(guessPool)
	}

	var wg sync.WaitGroup
	chunkSize := (len(guessPool) + numWorkers - 1) / numWorkers
	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(guessPool) {
			end = len(guessPool)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			scoreRange(start, end)
		}()
	}
	wg.Wait()

	// Sort: fewest worst-case guesses first, then fewest remaining candidates, then alphabetical
	slices.SortFunc(suggestions, func(a, b Suggestion) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		if a.Width != b.Width {
			return a.Width - b.Width
		}
		return strings.Compare(a.Word, b.Word)
	})

	return suggestions
}

func getValidWords(words []string, guesses []Guess) []string {
	result := make([]string, 0, len(words))
outer:
	for _, word := range words {
		for _, guess := range guesses {
			if word == string(guess.Word[:]) {
				continue outer
			}
			if !wordMatchesGuess(word, guess) {
				continue outer
			}
		}
		result = append(result, word)
	}
	return result
}
