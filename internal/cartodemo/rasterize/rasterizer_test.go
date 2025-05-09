package rasterize_test

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/GruffGemini/simplefeatures/geom"
	"github.com/GruffGemini/simplefeatures/internal/cartodemo/rasterize"
)

func TestRasterizer(t *testing.T) {
	const sz = 16
	rast := rasterize.NewRasterizer(sz, sz)

	ls, err := geom.UnmarshalWKT("LINESTRING(4 4, 12 8, 4 12)")
	expectNoErr(t, err)
	rast.LineString(ls.MustAsLineString())

	img := image.NewRGBA(image.Rect(0, 0, sz, sz))
	rast.Draw(img, img.Bounds(), image.NewUniform(color.Black), image.Point{})

	err = os.WriteFile("testdata/line.png", imageToPNG(t, img), 0o600)
	expectNoErr(t, err)
}
