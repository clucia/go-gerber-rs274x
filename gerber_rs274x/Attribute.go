package gerber_rs274x

import cairo "github.com/ungerik/go-cairo"

type Attribute struct {
	typ  string
	name string
	args []string
}

func (attrib Attribute) DataBlockPlaceholder() {

}

func (attrib Attribute) ProcessDataBlockBoundsCheck(imageBounds *ImageBounds, gfxState *GraphicsState) error {

	return nil
}

func (attrib Attribute) ProcessDataBlockToolpath(*CamOutput, *GraphicsState) error {
	return nil
}

func (attrib Attribute) ProcessDataBlockSurface(surface *cairo.Surface, gfxState *GraphicsState) error {
	return nil
}
