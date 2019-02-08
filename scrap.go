package scrap

/*
#cgo CFLAGS: -I${SRCDIR}/scrap-sys
#cgo LDFLAGS: -L${SRCDIR}/scrap-sys/target/release -lscrap_sys
#cgo windows LDFLAGS: -lws2_32 -luserenv -ldxgi -ld3d11

#include <stddef.h>
#include <scrap-sys.h>

struct Display* display_list_at(struct Display** list, int index) {
	return list[index];
}
*/
import "C"
import (
	"errors"
	"runtime"
)

func Displays() ([]*Display, error) {
	list := C.display_list()
	if list.err != nil {
		return nil, fromCgoErr(list.err)
	}
	ret := make([]*Display, list.len)
	for i := 0; i < len(ret); i++ {
		ret[i] = newDisplay(C.display_list_at(list.list, C.int(i)))
	}
	return ret, nil
}

type Display struct {
	cgoDisplay *C.struct_Display
	owned      bool
}

func newDisplay(d *C.struct_Display) *Display {
	display := &Display{cgoDisplay: d}
	display.setOwned(true)
	return display
}

func finalizeDisplay(d *Display) { C.display_free(d.cgoDisplay) }

func (d *Display) assertOwned() {
	if !d.owned {
		panic("Display not owned")
	}
}

func (d *Display) setOwned(owned bool) {
	if owned {
		if d.owned {
			panic("Already owned")
		}
		runtime.SetFinalizer(d, finalizeDisplay)
	} else {
		d.assertOwned()
		runtime.SetFinalizer(d, nil)
	}
	d.owned = owned
}

func PrimaryDisplay() (*Display, error) {
	d := C.display_primary()
	if d.err != nil {
		return nil, fromCgoErr(d.err)
	}
	return newDisplay(d.display), nil
}

func (d *Display) Width() int {
	d.assertOwned()
	return int(C.display_width(d.cgoDisplay))
}

func (d *Display) Height() int {
	d.assertOwned()
	return int(C.display_height(d.cgoDisplay))
}

type Capturer struct {
	cgoCapturer *C.struct_Capturer
}

func finalizeCapturer(c *Capturer) { C.capturer_free(c.cgoCapturer) }

func NewCapturer(display *Display) (*Capturer, error) {
	// Take ownership
	display.setOwned(false)
	c := C.capturer_new(display.cgoDisplay)
	if c.err != nil {
		return nil, fromCgoErr(c.err)
	}
	capturer := &Capturer{c.capturer}
	runtime.SetFinalizer(capturer, finalizeCapturer)
	return capturer, nil
}

func (c *Capturer) Width() int {
	return int(C.capturer_width(c.cgoCapturer))
}

func (c *Capturer) Height() int {
	return int(C.capturer_height(c.cgoCapturer))
}

func fromCgoErr(raw *C.char) error {
	defer C.error_free(raw)
	return errors.New(C.GoString(raw))
}
