package main

// Word list embedded from words.txt at compile time.

import (
	_ "embed"
	"strings"
)

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
