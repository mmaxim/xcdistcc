package ui

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"mmaxim.org/xcdistcc/client"
)

func NewMainWindow(a fyne.App, remotes []client.Remote) fyne.Window {
	w := a.NewWindow("Hello")
	refresher := NewRefresher(remotes)
	go func() {
		for {
			statuses, err := refresher.GetStatuses()
			if err != nil {
				log.Printf("failed to get status: %s", err)
			} else {
				vbox := container.NewVBox()
				for _, status := range statuses {
					vbox.Add(widget.NewLabel(status))
				}
				padded := container.NewPadded(vbox)
				w.SetContent(padded)
				w.Resize(fyne.NewSize(600.0, 400.0))
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	return w
}
