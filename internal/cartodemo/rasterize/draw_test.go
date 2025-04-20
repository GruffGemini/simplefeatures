package rasterize_test

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/GruffGemini/simplefeatures/geom"
	"github.com/GruffGemini/simplefeatures/internal/cartodemo/rasterize"
)

func TestDrawLine(t *testing.T) {
	g, err := geom.UnmarshalWKT("LINESTRING(4 4, 12 8, 4 12)")
	expectNoErr(t, err)

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	rast := rasterize.NewRasterizer(16, 16)
	rast.LineString(g.MustAsLineString())
	rast.Draw(img, img.Bounds(), image.NewUniform(color.Black), image.Point{})

	err = os.WriteFile("testdata/line.png", imageToPNG(t, img), 0o600)
	expectNoErr(t, err)
}
