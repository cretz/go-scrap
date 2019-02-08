package scrap

/*
#cgo CFLAGS: -I${SRCDIR}/scrap-sys
#cgo LDFLAGS: -L${SRCDIR}/scrap-sys/target/release -lscrap_sys
#cgo windows LDFLAGS: -lws2_32 -luserenv -ldxgi -ld3d11

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

func MakeDPIAware() error {
	if C.set_dpi_aware() == 0 {
		return errors.New("Failed setting DPI aware")
	}
	return nil
}

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
	// We cache these values since they may be referenced per frame
	width, height int
}

func finalizeCapturer(c *Capturer) { C.capturer_free(c.cgoCapturer) }

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

func (c *Capturer) Width() int {
	return c.width
}

func (c *Capturer) Height() int {
	return c.height
}

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
	runtime.SetFinalizer(img, finalizeFrameImage)
	return
}

// TODO: This fails right now...
func DisposeFrame(pix []uint8) {
	C.frame_free((*C.uchar)(unsafe.Pointer(&pix[0])), C.size_t(len(pix)))
}

type FrameImage struct {
	Pix                   []uint8
	Stride, Width, Height int
}

func finalizeFrameImage(f *FrameImage) {
	DisposeFrame(f.Pix)
}

func (f *FrameImage) ColorModel() color.Model { return color.RGBAModel }

func (f *FrameImage) Bounds() image.Rectangle { return image.Rect(0, 0, f.Width, f.Height) }

func (f *FrameImage) At(x, y int) color.Color { return f.RGBAAt(x, y) }

func (f *FrameImage) RGBAAt(x, y int) color.RGBA {
	if x < 0 || y < 0 || x > f.Width || y > f.Height {
		return color.RGBA{}
	}
	i := f.PixOffset(x, y)
	return color.RGBA{f.Pix[i+2], f.Pix[i+1], f.Pix[i], f.Pix[i+3]}
}

func (f *FrameImage) PixOffset(x, y int) int {
	return f.Stride*y + 4*x
}

func (f *FrameImage) Opaque() bool {
	// TODO: is there ever a case where this should be calculated instead?
	return true
}

func (f *FrameImage) ToRGBAImage() *image.RGBA {
	pixBGRA := f.Pix
	pixRGBA := make([]uint8, len(pixBGRA))
	// Just keep jumping by 4 rearranging colors
	// TODO: benchmark vs a copy followed by just swapping the B/R
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
