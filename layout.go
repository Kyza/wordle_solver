package main

import "fyne.io/fyne/v2"
import "fyne.io/fyne/v2/widget"

// tappableOverlay is an invisible widget that captures taps.
type tappableOverlay struct {
	widget.BaseWidget
	onTap func()
}

func newTappableOverlay(onTap func()) *tappableOverlay {
	t := &tappableOverlay{onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableOverlay) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableOverlay) CreateRenderer() fyne.WidgetRenderer {
	return &tappableOverlayRenderer{}
}

type tappableOverlayRenderer struct{}

func (r *tappableOverlayRenderer) Layout(_ fyne.Size)              {}
func (r *tappableOverlayRenderer) MinSize() fyne.Size              { return fyne.NewSize(0, 0) }
func (r *tappableOverlayRenderer) Refresh()                        {}
func (r *tappableOverlayRenderer) Objects() []fyne.CanvasObject    { return nil }
func (r *tappableOverlayRenderer) Destroy()                        {}

// fixedSizeLayout forces all children to a fixed size.
type fixedSizeLayout struct {
	width, height float32
}

func newFixedSizeLayout(w, h float32) *fixedSizeLayout {
	return &fixedSizeLayout{width: w, height: h}
}

func (l *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.width, l.height)
}

func (l *fixedSizeLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(l.width, l.height))
		o.Move(fyne.NewPos(0, 0))
	}
}
