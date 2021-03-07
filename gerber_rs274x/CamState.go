package gerber_rs274x

import "io"

type CamState GraphicsState

type CamOutput struct {
	wrt            io.WriteCloser
	toolWidth      float64
	x, y           float64
	power          int // laser power
	feedrate       int // laser power
	translateScale func(float64, float64) (float64, float64)
}

func NewCamOutput(
	wrt io.WriteCloser,
	feedrate int,
	toolWidth float64,
	x, y float64,
	power int,
	translateScale func(float64, float64) (float64, float64),
) *CamOutput {
	return &CamOutput{
		wrt:            wrt,
		toolWidth:      toolWidth,
		x:              x,
		y:              y,
		power:          power,
		feedrate:       feedrate,
		translateScale: translateScale,
	}
}
