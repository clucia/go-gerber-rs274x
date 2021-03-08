package main

import (
	"fmt"
	"os"

	"github.com/clucia/go-gerber-rs274x/gerber_rs274x"
)

type Layer struct {
	name  string
	typ   string
	fname string
	fext  string
}

var layers = []*Layer{
	{
		name:  "Front Copper",
		typ:   "COPPER",
		fname: "-F_Cu",
		fext:  ".gbr",
	},
	{
		name:  "Back Copper",
		typ:   "COPPER",
		fname: "-B_Cu",
		fext:  ".gbr",
	},
	{
		name:  "Edge Cuts",
		typ:   "EDGE",
		fname: "-Edge_Cuts",
		fext:  ".gbr",
	},
	{
		name:  "Drill",
		typ:   "DRILL",
		fname: "",
		fext:  ".drl",
	},
}

func main() {
	var err error

	var inFiles []*os.File

	if len(os.Args) != 2 {
		panic("usage")
	}
	fname := os.Args[1]

	bounds := gerber_rs274x.ImageBounds{}

	for _, ext := range layers {
		f, err := os.Open(fname + ext.fname + ext.fext)
		if err != nil {
			panic("open failure")
		}
		inFiles = append(inFiles, f)
	}

	// var gerberDBs = make([][]gerber_rs274x.DataBlock, len(inFiles))
	// var gerberDB []gerber_rs274x.DataBlock
	var ASTs = make([]interface{}, len(layers))
	var AST interface{}
	for i, ext := range layers {
		switch {
		case ext.typ == "COPPER":
			AST, err = gerber_rs274x.ParseGerberFile(inFiles[i])
			if err != nil {
				panic("gerber parse fail")
			}
			ASTs[i] = AST
			gerber_rs274x.GenerateBounds(AST.([]gerber_rs274x.DataBlock), &bounds)
		case ext.typ == "DRILL":
			drl := gerber_rs274x.NewDrlData()
			drl.ParseDrlFile(inFiles[3])

			ASTs[i] = drl

			dbounds := gerber_rs274x.NewDrillBounds()
			drl.GetBounds(dbounds)
			bounds.UpdateBounds(dbounds.Get())
		}
	}

	fmt.Println("bounds = ", bounds)
	xMin, _, yMin, _ := bounds.Get()
	tsFunc := func(x float64, y float64) (x0 float64, y0 float64) {
		// xMin, _, yMin, _ := bounds.Get()
		return x - xMin, y - yMin
	}

	var outputFile *os.File
	for i, ext := range layers {
		switch {
		case ext.typ == "COPPER":
			outputFile, err = os.Create(fname + layers[i].fname + ".gcode")
			if err != nil {
				panic("")
			}
			camo := gerber_rs274x.NewCamOutput(outputFile, 300, .2, 0, 0, 10, tsFunc)
			err := gerber_rs274x.GenerateToolpath(camo, ASTs[i].([]gerber_rs274x.DataBlock))
			if err != nil {
				fmt.Printf("Error generating toolpath file: %s\n", err.Error())
				os.Exit(5)
			}
		case ext.typ == "DRILL":

			outputFile, err = os.Create(fname + ".gcode")
			if err != nil {
				fmt.Printf("Error opening output file %s: %s\n", fname+".drl", err.Error())

				os.Exit(2)
			}
			camo := &gerber_rs274x.DrlCAM{Wrt: outputFile, ChangeZ: 15.0, SafeZ: 1.0, DrillZ: -3.0, DrillF: 20, TranslateScale: tsFunc}
			ASTs[i].(*gerber_rs274x.DrlData).GenGcode(camo)
			outputFile.Close()
		}
	}
}
