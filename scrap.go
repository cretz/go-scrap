/*
Package scrap is a Go wrapper around the Rust
https://github.com/quadrupleslap/scrap library. It supports reasonably fast
capturing of raw screen pixels. The library dependency is only at compile time
and statically compiled into the binary.

Since go-scrap statically links the Scrap library, the scrap-sys subdirectory
Rust project must be built in release mode before compiling this project. See
the README at https://github.com/cretz/go-scrap for more info.
*/
package scrap

/*
#cgo CFLAGS: -I${SRCDIR}/scrap-sys
#cgo LDFLAGS: -L${SRCDIR}/scrap-sys/target/release -lscrap_sys
#cgo windows LDFLAGS: -lws2_32 -luserenv -ldxgi -ld3d11
#cgo linux LDFLAGS: -ldl -lxcb -lxcb-shm -lxcb-randr

#include <stddef.h>
#include <scrap-sys.h>

#ifdef _WIN32
	#include <Windows.h>
	int set_dpi_aware() {
		return SetProcessDPIAware();
	}
#else
	int set_dpi_aware() {
		return 1;
	}
#endif

struct Display* display_list_at(struct Display** list, int index) {
	return list[index];
}
*/
import "C"
import (
	"errors"
	"image"
	"image/color"
	"runtime"
	"unsafe"
)

// MakeDPIAware enables the DPI aware setting for this process. This is
// currently only applicable for Windows. When DPI aware, the Width and Height
// of the Display and Capturer will return the full resolution for the screen
// instead of the scaled size.
func MakeDPIAware() error {
	if C.set_dpi_aware() == 0 {
		return errors.New("Failed setting DPI aware")
	}
	return nil
}

// Displays returns the set of known displays.
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

// Display represents a system display that can be captured. Once a display
// is used in NewCapturer, no other methods can be called on it.
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

// PrimaryDisplay returns the primary display of the system or an error.
func PrimaryDisplay() (*Display, error) {
	d := C.display_primary()
	if d.err != nil {
		return nil, fromCgoErr(d.err)
	}
	return newDisplay(d.display), nil
}

// Width gets the width of this display. This will panic if it is called after
// the display has been passed to NewCapturer.
func (d *Display) Width() int {
	d.assertOwned()
	return int(C.display_width(d.cgoDisplay))
}

// Height gets the height of this display. This will panic if it is called after
// the display has been passed to NewCapturer.
func (d *Display) Height() int {
	d.assertOwned()
	return int(C.display_height(d.cgoDisplay))
}

// Capturer represents the capturing of a display.
type Capturer struct {
	cgoCapturer *C.struct_Capturer
	// We cache these values since they may be referenced per frame
	width, height int
}

func finalizeCapturer(c *Capturer) { C.capturer_free(c.cgoCapturer) }

// NewCapturer creates a capturer for the given display. Methods on the display
// can no longer be called after passed to this function.
func NewCapturer(display *Display) (*Capturer, error) {
	// Take ownership
	display.setOwned(false)
	c := C.capturer_new(display.cgoDisplay)
	if c.err != nil {
		return nil, fromCgoErr(c.err)
	}
	capturer := &Capturer{
		cgoCapturer: c.capturer,
		width:       int(C.capturer_width(c.capturer)),
		height:      int(C.capturer_height(c.capturer)),
	}
	runtime.SetFinalizer(capturer, finalizeCapturer)
	return capturer, nil
}

// Width returns the width of this captured display.
func (c *Capturer) Width() int {
	return c.width
}

// Height returns the height of this captured display.
func (c *Capturer) Height() int {
	return c.height
}

// Frame gets an individual frame for this captured display. If an error occurs,
// it is returned. If capturing the frame would be a blocking call, wouldBlock
// is set to true and the pix is empty. Otherwise, if the frame is captured,
// wouldBlock is false and the error is nil.
//
// The resulting frame data is in packed BGRA format. This means that every
// pixel is represented by 4 values: blue, green, red, and alpha in that order.
// The "stride" is how many values are present in each row and is easily
// calculated as value count / height. For each row, there are at least 4 *
// width values for the BGRA representation, but there may be unused padding
// values at the end of the row.
//
// When a frame slice is returned, it is owned by the Capturer. It very likely
// will be overwritten by the next call to Frame. It also will be disposed of
// when the capturer is. The general rule is not to mutate the slice and don't
// store/use it beyond the lifetime of this Capturer.
func (c *Capturer) Frame() (pix []uint8, wouldBlock bool, err error) {
	f := C.capturer_frame(c.cgoCapturer)
	if f.err != nil {
		return nil, false, fromCgoErr(f.err)
	} else if f.would_block == 1 {
		return nil, true, nil
	}
	l := int(f.len)
	return (*[1 << 28]uint8)(unsafe.Pointer(f.data))[:l:l], false, nil
}

// FrameImage wraps the result of Frame into a FrameImage. It inherits the same
// ownership rules and restrictions of the Frame slice result.
func (c *Capturer) FrameImage() (img *FrameImage, wouldBlock bool, err error) {
	pix, wouldBlock, err := c.Frame()
	if wouldBlock || err != nil {
		return nil, wouldBlock, err
	}
	img = &FrameImage{
		Pix:    pix,
		Stride: len(pix) / c.height,
		Width:  c.width,
		Height: c.height,
	}
	return
}

// FrameImage is an implementation of image.Image. It carries the same ownership
// rules and restrictions as the Capturer.Frame slice result.
type FrameImage struct {
	// Pix is the raw slice of packed BGRA pixels. For more information on the
	// format and ownership rules and restrictions, see Capturer.Frame.
	Pix []uint8
	// Stride is the number of values that make up each vertical row. It is
	// simply len(Pix) / Height. See Capturer.Frame for more info.
	Stride int
	// Width is the width of the image.
	Width int
	// Height is the height of the image.
	Height int
}

var _ image.Image = &FrameImage{}

// ColorModel implements image.ColorModel.
func (f *FrameImage) ColorModel() color.Model { return color.RGBAModel }

// ColorModel implements image.Bounds.
func (f *FrameImage) Bounds() image.Rectangle { return image.Rect(0, 0, f.Width, f.Height) }

// At implements image.At.
func (f *FrameImage) At(x, y int) color.Color { return f.RGBAAt(x, y) }

// RGBAAt returns the RGBA color at the given point.
func (f *FrameImage) RGBAAt(x, y int) color.RGBA {
	if x < 0 || y < 0 || x > f.Width || y > f.Height {
		return color.RGBA{}
	}
	i := f.PixOffset(x, y)
	return color.RGBA{f.Pix[i+2], f.Pix[i+1], f.Pix[i], f.Pix[i+3]}
}

// PixOffset gives the index of the Pix where the 4-value BGRA pixel is.
func (f *FrameImage) PixOffset(x, y int) int {
	return f.Stride*y + 4*x
}

// Opaque always returns false as is present a performance optimization for
// algorithms such as PNG saving.
func (f *FrameImage) Opaque() bool {
	// TODO: is there ever a case where there is some transparency?
	return true
}

// ToRGBAImage converts this image to a image.RGBA image. This has value because
// in some packages such as image/draw and image/png, image.RGBA values are
// given special fast-path treatment. Note, this copies the entire Pix slice, so
// the same ownership rules and restrictions on this image do not apply to the
// result.
func (f *FrameImage) ToRGBAImage() *image.RGBA {
	pixBGRA := f.Pix
	pixRGBA := make([]uint8, len(pixBGRA))
	// Just keep jumping by 4 rearranging colors.
	// Note, benchmark showed there is no real difference between this and
	// copying then just swapping two.
	for i := 0; i < len(pixBGRA); i += 4 {
		pixRGBA[i] = pixBGRA[i+2]
		pixRGBA[i+1] = pixBGRA[i+1]
		pixRGBA[i+2] = pixBGRA[i]
		pixRGBA[i+3] = pixBGRA[i+3]
	}
	return &image.RGBA{Pix: pixRGBA, Stride: f.Stride, Rect: f.Bounds()}
}

func fromCgoErr(raw *C.char) error {
	defer C.error_free(raw)
	return errors.New(C.GoString(raw))
}
