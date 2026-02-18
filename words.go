package main

// Word list embedded from words.txt at compile time.
// Run "go generate" (or use the Makefile) to download the list before building.

import (
	_ "embed"
	"strings"
)

//go:generate curl -sSfL -o words.txt https://raw.githubusercontent.com/tabatkins/wordle-list/main/words

//go:embed words.txt
var wordsRaw string

func loadWords() []string {
	lines := strings.Split(strings.TrimSpace(wordsRaw), "\n")
	var words []string
	for _, w := range lines {
		w = strings.TrimSpace(strings.ToLower(w))
		if len(w) == 5 {
			words = append(words, w)
		}
	}
	return words
}
