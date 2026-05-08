package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("com.keenetic.tray.mobile")
	w := a.NewWindow("Keenetic Tray")
	w.Resize(fyne.NewSize(400, 700))

	ui := newMainUI(a, w)
	w.SetContent(ui.content())
	w.ShowAndRun()
}
