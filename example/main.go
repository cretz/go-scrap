package main

import (
	"fmt"

	"github.com/cretz/go-scrap"
)

func main() {
	displays, err := scrap.Displays()
	if err != nil {
		panic(err)
	}
	for _, display := range displays {
		fmt.Printf("WI: %v - HI: %v\n", display.Width(), display.Height())
	}
	d, err := scrap.PrimaryDisplay()
	if err != nil {
		panic(err)
	}
	fmt.Printf("WI: %v - %v\n", d, d.Width())
	c, err := scrap.NewCapturer(d)
	if err != nil {
		panic(err)
	}
	fmt.Printf("OH: %v - %v\n", c.Width(), c.Height())
}
