package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cretz/go-scrap"
)

func main() {
	fileName := "out.mp4"
	if len(os.Args) > 1 {
		fileName = os.Args[1]
	}
	// Record to video and wait for enter key asynchronously
	fmt.Printf("Starting...press enter to exit...")
	errCh := make(chan error, 2)
	ctx, cancelFn := context.WithCancel(context.Background())
	// Record
	go func() { errCh <- recordToVideo(ctx, fileName) }()
	// Wait for enter
	go func() {
		fmt.Scanln()
		errCh <- nil
	}()
	err := <-errCh
	cancelFn()
	if err != nil && err != context.Canceled {
		log.Fatalf("Execution failed: %v", err)
	}
	// Wait a bit...
	time.Sleep(4 * time.Second)
}

func recordToVideo(ctx context.Context, fileName string) error {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()
	// // Make DPI aware
	// if err := scrap.MakeDPIAware(); err != nil {
	// 	return err
	// }
	// Create the capturer
	cap, err := capturer()
	if err != nil {
		return err
	}
	// Build ffmpeg
	ffmpeg := exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "bgr0",
		"-video_size", fmt.Sprintf("%vx%v", cap.Width(), cap.Height()),
		"-i", "-",
		// "-vf", "scale=w=960:h=720:force_original_aspect_ratio=decrease",
		// "-filter", "minterpolate='fps=60'",
		"-c:v", "libx264", "-preset", "veryfast", //"-crf", "0",
		fileName,
	)
	// Stdin for sending data
	stdin, err := ffmpeg.StdinPipe()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	defer stdin.Close()
	// Run it in the background
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("Executing: %v\n", strings.Join(ffmpeg.Args, " "))
		out, err := ffmpeg.CombinedOutput()
		fmt.Printf("FFMPEG output:\n%v\n", string(out))
		errCh <- err
	}()
	// Just start sending a bunch of frames
	for {
		// Get the frame...
		if pix, _, err := cap.Frame(); err != nil {
			return err
		} else if pix != nil {
			// Send a row at a time
			stride := len(pix) / cap.Height()
			rowLen := 4 * cap.Width()
			for i := 0; i < len(pix); i += stride {
				if _, err = stdin.Write(pix[i : i+rowLen]); err != nil {
					break
				}
			}
			buf.Reset()
			print(".")
			// TODO: this is failing... scrap.DisposeFrame(pix)
			if err != nil {
				return err
			}
		}
		// Check if we're done, otherwise go again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		default:
		}
	}
}

func capturer() (*scrap.Capturer, error) {
	if d, err := scrap.PrimaryDisplay(); err != nil {
		return nil, err
	} else if c, err := scrap.NewCapturer(d); err != nil {
		return nil, err
	} else {
		return c, nil
	}
}
