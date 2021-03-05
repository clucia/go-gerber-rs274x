package main

import (
	"fmt"
	"os"
	"path/filepath"

	"../gerber_rs274x"
)

func main() {
	var inputFile *os.File
	var err error
	var fname string

	switch {
	case len(os.Args) == 1:
		fname = "/Users/chuck/Documents/electronics/test-amp-4/amp/amp-F_Cu.gbr"
	case len(os.Args) == 2:
		fname = os.Args[1]
	}
	inputFile, err = os.Open(fname)
	if err != nil {
		fmt.Printf("Error opening input file %s: %s\n", fname, err.Error())
		os.Exit(2)
	}
	if parsedFile, err := gerber_rs274x.ParseGerberFile(inputFile); err != nil {
		inputFile.Close()
		fmt.Printf("Error parsing gerber file: %v\n", err)
		os.Exit(3)
	} else {
		inputFile.Close()

		if parsedFile == nil {
			return //
		}
		outputFileName := filepath.Base(fname + ".gcode")

		f, err := os.Create(outputFileName)
		if err != nil {
			panic("")
		}

		camo := gerber_rs274x.NewCamOutput(f, .2, 0, 0, 5)

		if err := gerber_rs274x.GenerateToolpath(camo, parsedFile); err != nil {
			fmt.Printf("Error generating PNG file: %s\n", err.Error())
			os.Exit(5)
		}
	}
}
