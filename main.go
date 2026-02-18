package main

import (
	"fmt"
	"image/color"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	colorGray   = color.NRGBA{R: 120, G: 124, B: 126, A: 255}
	colorYellow = color.NRGBA{R: 201, G: 180, B: 88, A: 255}
	colorGreen  = color.NRGBA{R: 106, G: 170, B: 100, A: 255}
	colorEmpty  = color.NRGBA{R: 58, G: 58, B: 60, A: 255}
	colorWhite  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	colorCursor = color.NRGBA{R: 200, G: 200, B: 255, A: 255} // light blue border for selected cell
)

func clueColor(c Clue) color.Color {
	switch c {
	case Correct:
		return colorGreen
	case Misplaced:
		return colorYellow
	default:
		return colorGray
	}
}

var a = app.New()
var window fyne.Window
var gs *GameState

func main() {
	gs := NewGameState()

	window := a.NewWindow("Wordle Solver")
	window.Resize(fyne.Size{Width: 400, Height: 600})

	// Tile grid: 6 rows × 5 columns
	const tileSize float32 = 52
	const tilePad float32 = 4

	type tileUI struct {
		bg   *canvas.Rectangle
		text *canvas.Text
		btn  *tappableOverlay
	}

	var tileGrid [Rows][Cols]tileUI

	// refreshTile updates a single tile's appearance
	refreshTile := func(r, c int) {
		t := &tileGrid[r][c]
		letter, _ := gs.Letters[r][c].Get()
		clue, _ := gs.Clues[r][c].Get()
		isCursor := gs.IsCursor(r, c)

		t.text.Text = letter
		t.text.Refresh()

		if letter == "" {
			t.bg.FillColor = colorEmpty
		} else {
			t.bg.FillColor = clueColor(Clue(clue))
		}
		if isCursor {
			t.bg.StrokeColor = colorCursor
			t.bg.StrokeWidth = 3
		} else {
			t.bg.StrokeColor = color.NRGBA{R: 86, G: 87, B: 88, A: 255}
			t.bg.StrokeWidth = 2
		}
		t.bg.Refresh()
	}

	// refreshAllTiles refreshes every tile (used when cursor moves)
	refreshAllTiles := func() {
		for r := 0; r < Rows; r++ {
			for c := 0; c < Cols; c++ {
				refreshTile(r, c)
			}
		}
	}

	gridRows := make([]fyne.CanvasObject, Rows)
	for r := 0; r < Rows; r++ {
		row := r
		tileCols := make([]fyne.CanvasObject, Cols)
		for c := 0; c < Cols; c++ {
			col := c

			bg := canvas.NewRectangle(colorEmpty)
			bg.StrokeColor = color.NRGBA{R: 86, G: 87, B: 88, A: 255}
			bg.StrokeWidth = 2
			bg.CornerRadius = 4

			text := canvas.NewText("", colorWhite)
			text.TextSize = 24
			text.TextStyle = fyne.TextStyle{Bold: true}
			text.Alignment = fyne.TextAlignCenter

			btn := newTappableOverlay(func() {
				if gs.IsCursor(row, col) {
					// Already selected — cycle clue
					letter, _ := gs.Letters[row][col].Get()
					if letter != "" {
						gs.CycleClue(row, col)
						refreshTile(row, col)
					}
				} else {
					// Select this cell
					gs.SelectCell(row, col)
					refreshAllTiles()
				}
			})

			tileGrid[r][c] = tileUI{bg: bg, text: text, btn: btn}

			tile := container.NewStack(
				container.New(layout.NewMaxLayout(), bg),
				container.NewCenter(text),
				btn,
			)
			tileCols[c] = container.New(layout.NewCustomPaddedLayout(tilePad, tilePad, tilePad, tilePad),
				container.New(newFixedSizeLayout(tileSize, tileSize), tile),
			)

			// Listen to letter/clue changes
			gs.Letters[row][col].AddListener(binding.NewDataListener(func() {
				refreshTile(row, col)
			}))
			gs.Clues[row][col].AddListener(binding.NewDataListener(func() {
				refreshTile(row, col)
			}))
		}
		gridRows[r] = container.NewHBox(tileCols...)
	}

	// Listen to cursor changes to update borders
	gs.CursorRow.AddListener(binding.NewDataListener(func() { refreshAllTiles() }))
	gs.CursorCol.AddListener(binding.NewDataListener(func() { refreshAllTiles() }))

	boardGrid := container.NewVBox(gridRows...)

	// Keyboard handling
	window.Canvas().SetOnTypedRune(func(r rune) {
		if unicode.IsLetter(r) {
			gs.TypeLetter(r)
		}
	})
	window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyBackspace:
			gs.Backspace()
		case fyne.KeyEscape:
			gs.Deselect()
		case fyne.KeyReturn, fyne.KeyEnter:
			gs.Deselect()
			go gs.Solve()
		}
	})

	hardList := widget.NewListWithData(gs.HardSuggestions,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})
	hardList.OnSelected = func(id widget.ListItemID) {
		val, _ := gs.HardSuggestions.GetValue(id)
		gs.InsertWord(val)
		hardList.UnselectAll()
	}

	normalList := widget.NewListWithData(gs.NormalSuggestions,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})
	normalList.OnSelected = func(id widget.ListItemID) {
		val, _ := gs.NormalSuggestions.GetValue(id)
		gs.InsertWord(val)
		normalList.UnselectAll()
	}

	// Status label — bound to gs.StatusMessage
	statusLabel := widget.NewLabelWithData(gs.StatusMessage)
	statusLabel.Wrapping = fyne.TextWrapWord
	statusLabel.Alignment = fyne.TextAlignCenter

	// Solving overlay — spinner + progress bar
	spinner := widget.NewActivity()
	spinnerLabel := widget.NewLabelWithStyle("Solving...", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	progressBar := widget.NewProgressBarWithData(gs.Progress)
	progressBar.Min = 0
	progressBar.Max = 1
	spinnerBox := container.NewCenter(container.NewVBox(
		spinner,
		spinnerLabel,
		container.New(newFixedSizeLayout(300, 20), progressBar),
	))
	spinnerBox.Hide()

	// Main content
	resetButton := widget.NewButton("Reset", func() {
		go gs.Reset()
	})
	solveButton := widget.NewButton("Solve", func() {
		go gs.Solve()
	})

	mainContent := container.NewVSplit(
		container.NewVBox(
			container.NewCenter(boardGrid),
			statusLabel,
		),
		container.NewBorder(
			nil,
			container.NewGridWithColumns(2, resetButton, solveButton), nil, nil,
			container.NewHSplit(
				container.NewBorder(widget.NewLabelWithStyle("Normal Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), nil, nil, nil, normalList),
				container.NewBorder(widget.NewLabelWithStyle("Hard Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), nil, nil, nil, hardList),
			),
		))

	// Stack content + spinner overlay
	window.SetContent(container.NewStack(mainContent, spinnerBox))

	// Watch Solving state to show/hide spinner
	gs.Solving.AddListener(binding.NewDataListener(func() {
		solving, _ := gs.Solving.Get()
		if solving {
			progressBar.SetValue(0)
			mainContent.Hide()
			spinner.Start()
			spinnerBox.Show()
		} else {
			spinner.Stop()
			spinnerBox.Hide()
			mainContent.Show()
		}
	}))

	window.Show()
	a.Run()
	tidyUp()
}

func tidyUp() {
	fmt.Println("Exited")
}
