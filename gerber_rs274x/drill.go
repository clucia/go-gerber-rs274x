package gerber_rs274x

import (
	"bufio"
	"fmt"
	"io"
	"math"
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
	Wrt            io.WriteCloser
	ChangeZ        float64
	SafeZ          float64
	DrillZ         float64
	DrillF         int
	Tooln          int
	TranslateScale func(float64, float64) (float64, float64)
}

type DrillBounds struct {
	xMin float64
	xMax float64
	yMin float64
	yMax float64
}

func (db *DrillBounds) Get() (xMin, xMax, yMin, yMax float64) {
	xMin = db.xMin
	xMax = db.xMax
	yMin = db.yMin
	yMax = db.yMax
	return //
}

func (drl *DrlData) genDrillHole(cam *DrlCAM, st *Step) {
	w := cam.Wrt
	fmt.Fprintf(w, "G00Z%f\n", cam.SafeZ)
	x, y := cam.TranslateScale(st.x, st.y)
	fmt.Fprintf(w, "G00X%fY%f\n", x, y)
	fmt.Fprintf(w, "M3S10000\n")
	fmt.Fprintf(w, "G01Z%fF%d\n", cam.DrillZ, cam.DrillF)
	fmt.Fprintf(w, "G00Z%f\n", cam.SafeZ)
	fmt.Fprintf(w, "M5\n")
}

func (drl *DrlData) genChangeTool(cam *DrlCAM, st *Step) {
	w := cam.Wrt
	fmt.Fprintf(w, "G00Z%f\n", cam.ChangeZ)
	fmt.Fprintln(w, "G00X0Y0") // go home
	fmt.Fprintln(w, "M0")      // pause
	fmt.Fprintf(w, "G00Z%f\n", cam.SafeZ)
}

func (drl *DrlData) GenGcode(cam *DrlCAM) {
	fmt.Fprintf(cam.Wrt, "; My DrlCAM\n")
	fmt.Fprintf(cam.Wrt, "G90G40G17G21\n")
	for _, st := range drl.Steps {
		switch {
		case st.typ == "T":
			cam.Tooln = st.tooln
			drl.genChangeTool(cam, st)
		case st.typ == "D":
			drl.genDrillHole(cam, st)
		}
	}

}

func NewDrillBounds() *DrillBounds {
	return &DrillBounds{
		xMin: math.MaxFloat64,
		xMax: -math.MaxFloat64,
		yMin: math.MaxFloat64,
		yMax: -math.MaxFloat64,
	}
}
func (drl *DrlData) GetBounds(bounds *DrillBounds) {
	var tn int
	for _, st := range drl.Steps {
		switch {
		case st.typ == "T":
			tn = st.tooln
		case st.typ == "D":
			if tn >= len(drl.Tools) {
				panic("")
			}
			fmt.Println("st.x = ", st.x, ", st.y = ", st.y)
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

var rmComment *regexp.Regexp

func init() {
	rmComment = regexp.MustCompilePOSIX(`^(.*)[\t ]*[;](.*)`)
}

func (drl *DrlData) ParseDrlFile(rdr io.Reader) {
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
