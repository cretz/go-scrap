package scrap_test

import (
	"flag"
	"image/png"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cretz/go-scrap"
)

func TestMain(m *testing.M) {
	// Make DPI aware if requested
	dpiAware := flag.Bool("scrap.dpiaware", false, "Run w/ DPI awareness")
	flag.Parse()
	if *dpiAware {
		if err := scrap.MakeDPIAware(); err != nil {
			log.Panic(err)
		}
	}
	os.Exit(m.Run())
}

func Example_screenshot() {
	// Get the main display
	d, err := scrap.PrimaryDisplay()
	panicIfErr(err)
	// Create capturer for it
	c, err := scrap.NewCapturer(d)
	panicIfErr(err)
	// Get an image, trying until one available
	var img *scrap.FrameImage
	for img == nil {
		img, _, err = c.FrameImage()
		panicIfErr(err)
	}
	// Save it to PNG
	file, err := os.Create("screenshot.png")
	panicIfErr(err)
	defer file.Close()
	panicIfErr(png.Encode(file, img))
	// Output:
}

func TestFrameToRGBAImage(t *testing.T) {
	c, err := primaryScreenCapturer()
	failIfErr(t, err)
	var frameImg *scrap.FrameImage
	for frameImg == nil {
		frameImg, _, err = c.FrameImage()
		failIfErr(t, err)
	}
	rgbaImage := frameImg.ToRGBAImage()
	// Compare size
	if frameImg.Bounds() != rgbaImage.Bounds() {
		t.Fatal("Bounds mismatch")
	}
	// Compare every pixel
	for y := 0; y < frameImg.Height; y++ {
		for x := 0; x < frameImg.Width; x++ {
			if frameImg.At(x, y) != rgbaImage.At(x, y) {
				t.Fatal("Pixel mismatch")
			}
		}
	}
}

func BenchmarkRepeatedFrames(b *testing.B) {
	c, err := primaryScreenCapturer()
	failIfErr(b, err)
	// Just call first to warm it
	c.Frame()
	const framesToFetch = 2
	b.Run("Get frames, no sleep on block", func(b *testing.B) {
		for i := 0; i < framesToFetch; {
			pix, _, err := c.Frame()
			failIfErr(b, err)
			if len(pix) > 0 {
				i++
			}
		}
	})
	b.Run("Get frames, sleep on block", func(b *testing.B) {
		for i := 0; i < framesToFetch; {
			pix, _, err := c.Frame()
			failIfErr(b, err)
			if len(pix) > 0 {
				i++
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		}
	})
}

func BenchmarkFrameToRGBAImageApproaches(b *testing.B) {
	// We want to try out the two types to ToRGBAImage approaches to see which is faster.
	// RESULTS: difference is negligible.
	// First, get a copied set of pix.
	pixBGRA := func() []uint8 {
		c, err := primaryScreenCapturer()
		failIfErr(b, err)
		for {
			pix, _, err := c.Frame()
			failIfErr(b, err)
			if len(pix) > 0 {
				// Copy it off and return
				ret := make([]uint8, len(pix))
				copy(ret, pix)
				return ret
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	// Now do a couple of sub-benchmarks.
	b.Run("Loop and create each item", func(b *testing.B) {
		pixRGBA := make([]uint8, len(pixBGRA))
		for i := 0; i < len(pixBGRA); i += 4 {
			pixRGBA[i] = pixBGRA[i+2]
			pixRGBA[i+1] = pixBGRA[i+1]
			pixRGBA[i+2] = pixBGRA[i]
			pixRGBA[i+3] = pixBGRA[i+3]
		}
	})
	b.Run("Copy, then loop and just swap two", func(b *testing.B) {
		pixRGBA := make([]uint8, len(pixBGRA))
		copy(pixRGBA, pixBGRA)
		for i := 0; i < len(pixRGBA); i += 4 {
			pixRGBA[i], pixRGBA[i+2] = pixRGBA[i+2], pixRGBA[i]
		}
	})
}

func primaryScreenCapturer() (c *scrap.Capturer, err error) {
	d, err := scrap.PrimaryDisplay()
	if err == nil {
		c, err = scrap.NewCapturer(d)
	}
	return
}

func failIfErr(t interface{ Fatal(args ...interface{}) }, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
