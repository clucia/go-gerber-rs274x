package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
)

type Tool struct {
	typ  string
	size float64
}

// Step types:
//  T change tool
//  D Drill hole
type Step struct {
	typ   string
	tooln int
	x, y  float64
}

type DrlData struct {
	units string
	Tools []*Tool
	Steps []*Step
}

var defToolRe *regexp.Regexp
var changeToolRe *regexp.Regexp
var drillHoleRe *regexp.Regexp

func init() {
	defToolRe = regexp.MustCompilePOSIX(`^T([0-9]+)([A-Z])([0-9.-]+)`)
	changeToolRe = regexp.MustCompilePOSIX(`^T([0-9]+)$`)
	drillHoleRe = regexp.MustCompilePOSIX(`^X([0-9.-]+)Y([0-9.-]+)`)
}

func parseDrillHole(ln string) (step *Step, err error) {
	pdh := drillHoleRe.FindAllStringSubmatch(ln, -1)
	if len(pdh) == 0 {
		return nil, fmt.Errorf("bad directive")
	}
	x, err := strconv.ParseFloat(pdh[0][1], 64)
	if err != nil {
		return //
	}
	y, err := strconv.ParseFloat(pdh[0][2], 64)
	if err != nil {
		return //
	}
	step = &Step{
		typ: "D",
		x:   x,
		y:   y,
	}
	return //
}

func parseChangeTool(ln string) (step *Step, err error) {
	pct := changeToolRe.FindAllStringSubmatch(ln, -1)
	if len(pct) == 0 {
		return nil, fmt.Errorf("bad directive")
	}
	tooln, err := strconv.Atoi(pct[0][1])
	if err != nil {
		return //
	}
	step = &Step{
		typ:   "T",
		tooln: tooln,
	}
	return //
}

func parseDefineTool(ln string) (tool *Tool, tooln int, err error) {
	pdt := defToolRe.FindAllStringSubmatch(ln, -1)
	if len(pdt) == 0 {
		return nil, -1, fmt.Errorf("bad directive")
	}
	tooln, err = strconv.Atoi(pdt[0][1])
	if err != nil {
		return //
	}
	tool = &Tool{
		typ: pdt[0][2],
	}
	tool.size, err = strconv.ParseFloat(pdt[0][3], 64)
	return //
}

//			tool, tooln := parseTool(ln)

func NewDrlData() *DrlData {
	return &DrlData{}

}

type DrlCAM struct {
	wrt            io.WriteCloser
	changeZ        float64
	safeZ          float64
	drillZ         float64
	drillF         int
	tooln          int
	translateScale func(float64, float64) (float64, float64)
}

type DrillBounds struct {
	xMin float64
	xMax float64
	yMin float64
	yMax float64
}

func (drl *DrlData) genDrillHole(cam *DrlCAM, st *Step) {
	w := cam.wrt
	fmt.Fprintf(w, "G00Z%f\n", cam.safeZ)
	x, y := cam.translateScale(st.x, st.y)
	fmt.Fprintf(w, "G00X%fY%f\n", x, y)
	fmt.Fprintf(w, "M3S10000\n")
	fmt.Fprintf(w, "G01Z%fF%d\n", cam.drillZ, cam.drillF)
	fmt.Fprintf(w, "G00Z%f\n", cam.safeZ)
	fmt.Fprintf(w, "M5\n")
}

func (drl *DrlData) genChangeTool(cam *DrlCAM, st *Step) {
	w := cam.wrt
	fmt.Fprintf(w, "G00Z%f\n", cam.changeZ)
	fmt.Fprintln(w, "G00X0Y0") // go home
	fmt.Fprintln(w, "M0")      // pause
	fmt.Fprintf(w, "G00Z%f\n", cam.safeZ)
}

func (drl *DrlData) genGcode(cam *DrlCAM) {
	fmt.Fprintf(cam.wrt, "; My DrlCAM\n")
	fmt.Fprintf(cam.wrt, "G90G40G17G21\n")
	for _, st := range drl.Steps {
		switch {
		case st.typ == "T":
			cam.tooln = st.tooln
			drl.genChangeTool(cam, st)
		case st.typ == "D":
			drl.genDrillHole(cam, st)
		}
	}

}

func (drl *DrlData) getBounds(bounds *DrillBounds) {
	var tn int
	for _, st := range drl.Steps {
		switch {
		case st.typ == "T":
			tn = st.tooln
		case st.typ == "D":
			if tn >= len(drl.Tools) {
				panic("")
			}
			xMin := st.x - drl.Tools[tn].size/2.0
			xMax := st.x + drl.Tools[tn].size/2.0
			yMin := st.y - drl.Tools[tn].size/2.0
			yMax := st.y + drl.Tools[tn].size/2.0

			if xMin < bounds.xMin {
				bounds.xMin = xMin
			}
			if xMax > bounds.xMax {
				bounds.xMax = xMax
			}
			if yMin < bounds.yMin {
				bounds.yMin = yMin
			}
			if yMax > bounds.yMax {
				bounds.yMax = yMax
			}
		}
	}
}

func main() {
	var inputFile *os.File
	var err error
	var fname string

	switch {
	case len(os.Args) == 1:
		fname = "/Users/chuck/Documents/electronics/testPCB/testPCB.drl"
	case len(os.Args) == 2:
		fname = os.Args[1]
	}
	inputFile, err = os.Open(fname)
	if err != nil {
		fmt.Printf("Error opening input file %s: %s\n", fname, err.Error())
		os.Exit(2)
	}
	drl := NewDrlData()
	drl.parseDrlFile(inputFile)

	bounds := DrillBounds{
		xMin: math.MaxFloat64,
		xMax: -math.MaxFloat64,
		yMin: math.MaxFloat64,
		yMax: -math.MaxFloat64,
	}

	drl.getBounds(&bounds)
	fmt.Println(bounds)

	var outputFile *os.File
	outputFile, err = os.Create(fname + ".gcode")
	if err != nil {
		fmt.Printf("Error opening output file %s: %s\n", fname+".gcode", err.Error())
		os.Exit(2)
	}
	camo := &DrlCAM{
		wrt:     outputFile,
		changeZ: 15.0,
		safeZ:   1.0,
		drillZ:  -3.0,
		drillF:  20,
		translateScale: func(x float64, y float64) (x0 float64, y0 float64) {
			return x - bounds.xMin, y - bounds.yMin
		},
	}
	drl.genGcode(camo)
	outputFile.Close()

}

var rmComment *regexp.Regexp

func init() {
	rmComment = regexp.MustCompilePOSIX(`^(.*)[\t ]*[;](.*)`)
}

func (drl *DrlData) parseDrlFile(rdr io.Reader) {
	brdr := bufio.NewReader(rdr)

	for {
		line, pre, err := brdr.ReadLine()
		if err == io.EOF {
			return
		}
		if pre || err != nil {
			panic("")
		}
		ln := string(line)
		parsedLn := rmComment.FindAllStringSubmatch(ln, -1)
		fmt.Println("ln = ", ln, ", parsedLn = ", parsedLn)
		if len(parsedLn) > 0 {
			ln = parsedLn[0][1]
		}
		switch {
		case ln == "":
			continue
		case ln == "%":
			continue
		case ln == "G05":
			continue
		case ln == "G90":
			continue
		case ln == "M48":
		case ln == "M30":
		case ln == "FMAT,2":
		case ln == "INCH":
			fallthrough
		case ln == "METRIC":
			drl.units = ln
		case ln[0] == 'T':
			tool, tooln, err := parseDefineTool(ln)
			if err == nil {
				for i := len(drl.Tools); i <= tooln; i++ {
					drl.Tools = append(drl.Tools, nil)
				}
				drl.Tools[tooln] = tool
			} else {
				var step *Step
				step, err = parseChangeTool(ln)
				if err != nil {
					panic("")
				}
				fmt.Println("tooln = ", tooln)
				drl.Steps = append(drl.Steps, step)
			}
		case ln[0] == 'X':
			var step *Step
			step, err = parseDrillHole(ln)
			if err != nil {
				panic("")
			}
			drl.Steps = append(drl.Steps, step)
		default:
			panic("Unparsable")
		}
	}
}

/*
M48
; DRILL file {KiCad (5.99.0-8821-g31003c6e09)} date Saturday, March 06, 2021 at 01:26:26 PM
; FORMAT={-:-/ absolute / metric / decimal}
; #@! TF.CreationDate,2021-03-06T13:26:26-07:00
; #@! TF.GenerationSoftware,Kicad,Pcbnew,(5.99.0-8821-g31003c6e09)
FMAT,2
METRIC
T1C0.800
T2C1.000
T3C1.500
%
G90
G05
T1
X-187.757Y-324.307
X-187.731Y-306.553
T2
X-187.757Y-312.649
X-187.757Y-315.189
X-187.757Y-317.729
T3
X-206.12Y-314.681
X-199.12Y-314.681
T0
M30
*/
