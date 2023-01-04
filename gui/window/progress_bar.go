package window

import (
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func createProgressBar() *gtk.Scale {
	scale, err := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, 1.0, 0.01)
	handleError(err)

	scale.SetDrawValue(false)

	duration := 0.0
	expectedValue := 0.0

	updateValue := func() {
		position, err := playerInstance.GetPosition()
		if err != nil {
			return
		}

		expectedValue = position / duration
		glib.IdleAdd(func() {
			scale.SetValue(expectedValue)
		})
	}

	onUpdate(func(metadata map[string]dbus.Variant) {
		duration, err = playerInstance.GetLength()
		if err == nil {
			updateValue()
		}
	})

	scale.Connect("value-changed", func() {
		value := scale.GetValue()
		if value != expectedValue {
			playerInstance.SetPosition(value * duration)
		}
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			updateValue()
		}
	}()

	return scale
}
