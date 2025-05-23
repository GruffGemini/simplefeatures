package geom_test

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/GruffGemini/simplefeatures/geom"
)

// regularPolygon computes a regular polygon circumscribed by a circle with the
// given center and radius. Sides must be at least 3 or it will panic.
func regularPolygon(center geom.XY, radius float64, sides int) geom.Polygon {
	if sides <= 2 {
		panic(sides)
	}
	coords := make([]float64, 2*(sides+1))
	for i := 0; i < sides; i++ {
		angle := math.Pi/2 + float64(i)/float64(sides)*2*math.Pi
		coords[2*i+0] = center.X + math.Cos(angle)*radius
		coords[2*i+1] = center.Y + math.Sin(angle)*radius
	}
	coords[2*sides+0] = coords[0]
	coords[2*sides+1] = coords[1]
	ring := geom.NewLineString(geom.NewSequence(coords, geom.DimXY))
	return geom.NewPolygon([]geom.LineString{ring})
}

func BenchmarkMarshalWKB(b *testing.B) {
	b.Run("polygon", func(b *testing.B) {
		for _, sz := range []int{10, 100, 1000, 10000} {
			b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
				poly := regularPolygon(geom.XY{}, 1.0, sz)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					poly.AsBinary()
				}
			})
		}
	})
}

func BenchmarkUnmarshalWKB(b *testing.B) {
	b.Run("polygon", func(b *testing.B) {
		for _, sz := range []int{10, 100, 1000, 10000} {
			b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
				wkb := regularPolygon(geom.XY{}, 1.0, sz).AsBinary()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := geom.UnmarshalWKB(wkb, geom.NoValidate{})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}

func BenchmarkIntersectsLineStringWithLineString(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			var floats1, floats2 []float64
			for i := 0; i < sz; i++ {
				x := float64(i) / float64(sz)
				floats1 = append(floats1, x, 1)
				floats2 = append(floats2, x, 2)
			}
			seq1 := geom.NewSequence(floats1, geom.DimXY)
			seq2 := geom.NewSequence(floats2, geom.DimXY)
			ls1 := geom.NewLineString(seq1).AsGeometry()
			ls2 := geom.NewLineString(seq2).AsGeometry()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if geom.Intersects(ls1, ls2) {
					b.Fatal("should not intersect")
				}
			}
		})
	}
}

func BenchmarkIntersectsMultiPointWithMultiPoint(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", 2*sz), func(b *testing.B) {
			rnd := rand.New(rand.NewSource(0))
			var pointsA, pointsB []geom.Point
			for i := 0; i < sz; i++ {
				ptA := geom.XY{X: rnd.Float64(), Y: rnd.Float64()}.AsPoint()
				pointsA = append(pointsA, ptA)
				ptB := geom.XY{X: rnd.Float64(), Y: rnd.Float64()}.AsPoint()
				pointsB = append(pointsB, ptB)
			}
			mpA := geom.NewMultiPoint(pointsA).AsGeometry()
			mpB := geom.NewMultiPoint(pointsB).AsGeometry()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if geom.Intersects(mpA, mpB) {
					b.Fatal("shouldn't have intersected")
				}
			}
		})
	}
}

func BenchmarkPolygonSingleRingValidation(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			floats := make([]float64, 2*(sz+1))
			for i := 0; i < sz; i++ {
				angle := float64(i) / float64(sz) * 2 * math.Pi
				floats[2*i+0] = math.Cos(angle)
				floats[2*i+1] = math.Sin(angle)
			}
			floats[2*sz+0] = floats[0]
			floats[2*sz+1] = floats[1]
			ring := geom.NewLineString(geom.NewSequence(floats, geom.DimXY))
			poly := geom.NewPolygon([]geom.LineString{ring})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := poly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPolygonMultipleRingsValidation(b *testing.B) {
	for _, sz := range []int{2, 6, 20, 64} {
		b.Run(fmt.Sprintf("n=%d", sz*sz), func(b *testing.B) {
			rnd := rand.New(rand.NewSource(0))
			rings := make([]geom.LineString, sz*sz+1)
			rings[0] = geom.NewLineString(geom.NewSequence([]float64{0, 0, 0, 1, 1, 1, 1, 0, 0, 0}, geom.DimXY))
			for i := 0; i < sz*sz; i++ {
				center := geom.XY{
					X: (0.5 + float64(i/sz)) / float64(sz),
					Y: (0.5 + float64(i%sz)) / float64(sz),
				}
				dx := rnd.Float64() * 0.5 / float64(sz)
				dy := rnd.Float64() * 0.5 / float64(sz)
				rings[1+i] = geom.NewLineString(geom.NewSequence([]float64{
					center.X - dx, center.Y - dy,
					center.X + dx, center.Y - dy,
					center.X + dx, center.Y + dy,
					center.X - dx, center.Y + dy,
					center.X - dx, center.Y - dy,
				}, geom.DimXY))
			}
			poly := geom.NewPolygon(rings)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := poly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPolygonZigZagRingsValidation(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			outerRingEnv := geom.NewEnvelope(geom.XY{}, geom.XY{7, float64(sz + 1)})
			outerRing := outerRingEnv.AsGeometry().MustAsPolygon().ExteriorRing()
			var leftFloats, rightFloats []float64
			for i := 0; i < sz; i++ {
				leftFloats = append(leftFloats, float64(2+(i%2)*2), float64(1+i))
				rightFloats = append(rightFloats, float64(3+(i%2)*2), float64(1+i))
			}
			leftFloats = append(leftFloats,
				1, float64(sz),
				1, 1,
				2, 1,
			)
			rightFloats = append(rightFloats,
				6, float64(sz),
				6, 1,
				3, 1,
			)
			leftRing := geom.NewLineString(geom.NewSequence(leftFloats, geom.DimXY))
			rightRing := geom.NewLineString(geom.NewSequence(rightFloats, geom.DimXY))
			poly := geom.NewPolygon([]geom.LineString{outerRing, leftRing, rightRing})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := poly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPolygonAnnulusValidation(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			outer := regularPolygon(geom.XY{}, 1.0, sz/2).ExteriorRing()
			inner := regularPolygon(geom.XY{}, 0.5, sz/2).ExteriorRing()
			rings := []geom.LineString{outer, inner}
			poly := geom.NewPolygon(rings)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := poly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMultipolygonValidation(b *testing.B) {
	for _, sz := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("n=%d", sz*sz), func(b *testing.B) {
			rnd := rand.New(rand.NewSource(0))
			polys := make([]geom.Polygon, sz*sz)
			for i := 0; i < sz*sz; i++ {
				cx := (0.5 + float64(i/sz)) / float64(sz)
				cy := (0.5 + float64(i%sz)) / float64(sz)
				dx := rnd.Float64() * 0.5 / float64(sz)
				dy := rnd.Float64() * 0.5 / float64(sz)
				ring := geom.NewLineString(geom.NewSequence([]float64{
					cx - dx, cy - dy,
					cx + dx, cy - dy,
					cx + dx, cy + dy,
					cx - dx, cy + dy,
					cx - dx, cy - dy,
				}, geom.DimXY))
				polys[i] = geom.NewPolygon([]geom.LineString{ring})
			}
			multiPoly := geom.NewMultiPolygon(polys)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := multiPoly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMultiPolygonTwoCircles(b *testing.B) {
	for _, sz := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			const eps = 0.1
			polys := []geom.Polygon{
				regularPolygon(geom.XY{X: -eps, Y: -eps}, 1.0, sz),
				regularPolygon(geom.XY{X: math.Sqrt2, Y: math.Sqrt2}, 1.0, sz),
			}
			multiPoly := geom.NewMultiPolygon(polys)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := multiPoly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMultiPolygonMultipleTouchingPoints(b *testing.B) {
	for _, sz := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			fs1 := []float64{0, 0}
			fs2 := []float64{4, 0}
			for i := 0; i < 2*sz+1; i++ {
				fs1 = append(fs1, float64(1+i%2), float64(i))
				fs2 = append(fs2, float64(3-i%2), float64(i))
			}
			fs1 = append(fs1, 0, float64(2*sz), 0, 0)
			fs2 = append(fs2, 4, float64(2*sz), 4, 0)

			ls1 := geom.NewLineString(geom.NewSequence(fs1, geom.DimXY))
			ls2 := geom.NewLineString(geom.NewSequence(fs2, geom.DimXY))
			p1 := geom.NewPolygon([]geom.LineString{ls1})
			p2 := geom.NewPolygon([]geom.LineString{ls2})
			polys := []geom.Polygon{p1, p2}
			multiPoly := geom.NewMultiPolygon(polys)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := multiPoly.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkWKTParsing(b *testing.B) {
	for _, tc := range []struct {
		desc string
		wkt  string
	}{
		{
			"point",
			"POINT(-3.14159265359 3.14159265359)",
		},
	} {
		b.Run(tc.desc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := geom.UnmarshalWKT(tc.wkt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDistancePolygonToPolygonOrdering(b *testing.B) {
	for _, sz := range []int{100, 1000} {
		for _, swap := range []bool{false, true} {
			b.Run(fmt.Sprintf("n=%d_swap=%t", sz, swap), func(b *testing.B) {
				p1 := regularPolygon(geom.XY{0, 0}, 1.0, sz/10).AsGeometry()
				p2 := regularPolygon(geom.XY{3, 0}, 1.0, sz).AsGeometry()
				if swap {
					p1, p2 = p2, p1
				}
				for i := 0; i < b.N; i++ {
					geom.Distance(p1, p2)
				}
			})
		}
	}
}

func BenchmarkIntersectionPolygonWithPolygonOrdering(b *testing.B) {
	for _, sz := range []int{100, 1000} {
		for _, swap := range []bool{false, true} {
			b.Run(fmt.Sprintf("n=%d_swap=%t", sz, swap), func(b *testing.B) {
				p1 := regularPolygon(geom.XY{0, 0}, 1.0, sz/10).AsGeometry()
				p2 := regularPolygon(geom.XY{1, 0}, 1.0, sz).AsGeometry()
				if swap {
					p1, p2 = p2, p1
				}
				for i := 0; i < b.N; i++ {
					geom.Distance(p1, p2)
				}
			})
		}
	}
}

func BenchmarkMultiLineStringIsSimpleManyLineStrings(b *testing.B) {
	for _, sz := range []int{100, 1000} {
		b.Run(fmt.Sprintf("n=%d", sz), func(b *testing.B) {
			var lss []geom.LineString
			for i := 0; i < sz; i++ {
				seq := geom.NewSequence([]float64{
					float64(2*i + 0),
					float64(2*i + 0),
					float64(2*i + 1),
					float64(2*i + 1),
				}, geom.DimXY)
				ls := geom.NewLineString(seq)
				if err := ls.Validate(); err != nil {
					b.Fatal(err)
				}
				lss = append(lss, ls)
			}
			mls := geom.NewMultiLineString(lss)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mls.IsSimple()
			}
		})
	}
}

func BenchmarkForceCWandForceCCW(b *testing.B) {
	for i, tc := range []struct {
		wkt     string
		geoType geom.GeometryType
		isCW    bool
		isCCW   bool
		note    string
	}{
		{"POLYGON((0 0,0 5,5 5,5 0,0 0))", geom.TypePolygon, true, false, "CW"},
		{"POLYGON((1 1,3 1,2 2,2 4,1 1))", geom.TypePolygon, false, true, "CCW"},
		{"POLYGON((0 0,0 5,5 5,5 0,0 0), (1 1,3 1,2 2,2 4,1 1))", geom.TypePolygon, true, false, "outer CW inner CCW"},
		{"POLYGON((0 0,5 0,5 5,0 5,0 0), (1 1,1 2,2 2,2 1,1 1))", geom.TypePolygon, false, true, "outer CCW inner CW"},
		{"MULTIPOLYGON(((40 40, 45 30, 20 45, 40 40)),((20 35, 45 20, 30 5, 10 10, 10 30, 20 35),(30 20, 20 25, 20 15, 30 20)))", geom.TypeMultiPolygon, true, false, "all CW"},
		{"MULTIPOLYGON(((40 40, 20 45, 45 30, 40 40)),((20 35, 10 30, 10 10, 30 5, 45 20, 20 35),(30 20, 20 15, 20 25, 30 20)))", geom.TypeMultiPolygon, false, true, "all CCW"},
		{"GEOMETRYCOLLECTION(POLYGON((0 0,0 5,5 5,5 0,0 0)), MULTIPOLYGON(((40 40, 45 30, 20 45, 40 40)),((20 35, 45 20, 30 5, 10 10, 10 30, 20 35),(30 20, 20 25, 20 15, 30 20))))", geom.TypeGeometryCollection, true, false, "all CW"},
	} {
		g := geomFromWKT(b, tc.wkt)
		for _, correct := range map[string]bool{
			"correct":   true,
			"incorrect": false,
		} {
			b.Run(strconv.Itoa(i), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if tc.isCW && correct || tc.isCCW && !correct {
						g.ForceCW()
					} else if tc.isCCW && correct || tc.isCW && !correct {
						g.ForceCCW()
					}
				}
			})
		}
	}
}
