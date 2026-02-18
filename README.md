# Wordle Solver

A desktop Wordle solver built with Go and [Fyne](https://fyne.io/). Type in your guesses, set the clue colors, and get optimal next-guess suggestions ranked by worst-case performance.

## Features

- **Interactive tile grid** — type letters and click tiles to cycle clue colors (gray → yellow → green)
- **Partition-based solver** — uses recursive minimax to find guesses that minimize worst-case remaining candidates
- **Hard mode & normal mode** — hard mode only suggests valid candidates; normal mode considers all words
- **Parallel execution** — scales across all CPU cores
- **Adaptive search depth** — deeper analysis for smaller candidate pools (depth 3 under 50 candidates, depth 2 under 300, depth 1 otherwise)

## Screenshots

![In Progress](screenshots/in_progress.png)
![Solving](screenshots/solving.png)
![Solved](screenshots/solved.png)

## How to Use

1. Type a word into the grid (or click a suggestion to insert it)
2. Click filled tiles to cycle their clue colors to match Wordle's feedback
3. Press **Enter** or click **Solve**
4. Pick a suggestion from the **Normal Mode** or **Hard Mode** lists
5. Repeat until solved

### Reading Suggestions

- `SALET  ⌊3⌋  290 left` — worst-case depth of 3, largest partition has 290 candidates
- `CRANE ✓` — only one candidate remains; this is the answer

## Building

```sh
go generate ./...
go build -o wordle-solver .
./wordle-solver
```

`go generate` downloads the word list from the [tabatkins/wordle-list](https://github.com/tabatkins/wordle-list) repo. Requires `curl`.

Requires Go 1.21+ and a C compiler (for Fyne's OpenGL bindings).

## Running Tests

```sh
go test ./...
```

## Word List

The solver uses a ~14,855-word list downloaded at build time from [tabatkins/wordle-list](https://github.com/tabatkins/wordle-list) and embedded into the binary.
