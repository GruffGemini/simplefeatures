package geom_test

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/GruffGemini/simplefeatures/geom"
)

// TWKBTestCases is outside a single test function to allow multiple
// test functions to use it, and is exported to potentially allow
// automated testing from external test frameworks.
var TWKBTestCases = []struct {
	description            string
	twkbHex, wkt, postGIS  string
	hasZ, hasM             bool
	precXY, precZ, precM   int
	hasSize                bool
	hasBBox                bool
	listedBBox             [2]string
	hasIDList              bool
	listedIDs              []int64
	closeRings             bool
	skipDecode, skipEncode bool
}{
	// Several test cases adapted from https://github.com/TWKB/twkb.js/blob/master/test/twkb.spec.js
	{
		description: "point lacking data",
		twkbHex:     "0110",
		wkt:         "POINT EMPTY",
	},
	{
		description: "point",
		twkbHex:     "01000204",
		wkt:         "POINT(1 2)",
	},
	{
		description: "point z",
		twkbHex:     "010801020406",
		wkt:         "POINT Z (1 2 3)",
		hasZ:        true,
	},
	{
		description: "point m",
		twkbHex:     "010802020408",
		wkt:         "POINT M (1 2 4)",
		hasM:        true,
	},
	{
		description: "point zm",
		twkbHex:     "01080302040608",
		wkt:         "POINT ZM (1 2 3 4)",
		hasZ:        true,
		hasM:        true,
	},
	{
		description: "point with prec xy -1",
		twkbHex:     "11000204",
		wkt:         "POINT(10 20)",
		precXY:      -1,
	},
	{
		description: "point with prec xy 1",
		twkbHex:     "21000204",
		wkt:         "POINT(0.1 0.2)",
		precXY:      1,
	},
	{
		description: "point with prec xy -2",
		twkbHex:     "31000204",
		wkt:         "POINT(100 200)",
		precXY:      -2,
	},
	{
		description: "point with default prec but larger numbers",
		twkbHex:     "0100c8019003",
		wkt:         "POINT(100 200)",
		precXY:      0,
	},
	{
		description: "point with prec 7",
		twkbHex:     "e100cfddf89107b0a5e9d702",
		wkt:         "POINT (-95.8338920 36.0524120)",
		precXY:      7,
	},
	{
		description: "point with prec 7 requiring encoding (requires rounding)",
		twkbHex:     "e100cfddf89107b0a5e9d702",
		wkt:         "POINT (-95.83389199999999 36.052412)",
		precXY:      7,
		skipDecode:  true,
	},
	{
		description: "point with prec xy 2",
		twkbHex:     "41000204",
		wkt:         "POINT(0.01 0.02)",
		precXY:      2,
	},
	{
		description: "line string lacking data ",
		twkbHex:     "0210",
		wkt:         "LINESTRING EMPTY",
	},
	{
		description: "line string no points",
		twkbHex:     "020000",
		wkt:         "LINESTRING EMPTY",
		skipEncode:  true,
	},
	{
		description: "line string",
		twkbHex:     "02000202020808",
		wkt:         "LINESTRING(1 1,5 5)",
	},
	{
		description: "line string z",
		twkbHex:     "02080102020202080808",
		wkt:         "LINESTRING Z(1 1 1,5 5 5)",
		hasZ:        true,
	},
	{
		description: "line string z with prec xy -1 & prec z 1",
		twkbHex:     "12080502020202080808",
		wkt:         "LINESTRING Z(10 10 0.1,50 50 0.5)",
		hasZ:        true,
		precXY:      -1,
		precZ:       1,
	},
	{
		description: "line string z with prec xy 1 & prec z 2",
		twkbHex:     "22080902020202080808",
		wkt:         "LINESTRING Z(0.1 0.1 0.01,0.5 0.5 0.05)",
		hasZ:        true,
		precXY:      1,
		precZ:       2,
	},
	{
		description: "line string m with prec xy 2 & prec m 3",
		twkbHex:     "42086202020202080808",
		wkt:         "LINESTRING M(0.01 0.01 0.001,0.05 0.05 0.005)",
		hasM:        true,
		precXY:      2,
		precM:       3,
	},
	{
		description: "polygon lacking data",
		twkbHex:     "0310",
		wkt:         "POLYGON EMPTY",
	},
	{
		description: "polygon no rings",
		twkbHex:     "030000",
		wkt:         "POLYGON EMPTY",
		skipEncode:  true,
	},
	{
		description: "polygon unclosed rings",
		twkbHex:     "030002040000060000060500040203000202000001",
		wkt:         "POLYGON((0 0,3 0,3 3,0 3,0 0),(1 1,1 2,2 2,2 1,1 1))",
	},
	{
		description: "polygon closed rings",
		twkbHex:     "03000205000006000006050000050502020002020000010100",
		wkt:         "POLYGON((0 0,3 0,3 3,0 3,0 0),(1 1,1 2,2 2,2 1,1 1))",
		closeRings:  true,
	},
	{
		description: "polygon unclosed rings with size & bbox",
		twkbHex:     "0303170006000602040000060000060500040203000202000001",
		wkt:         "POLYGON((0 0,3 0,3 3,0 3,0 0),(1 1,1 2,2 2,2 1,1 1))",
		hasSize:     true,
		hasBBox:     true,
		listedBBox:  [2]string{"POINT(0 0)", "POINT(3 3)"},
	},
	{
		description: "polygon closed rings with size & bbox",
		twkbHex:     "03031b000600060205000006000006050000050502020002020000010100",
		wkt:         "POLYGON((0 0,3 0,3 3,0 3,0 0),(1 1,1 2,2 2,2 1,1 1))",
		hasSize:     true,
		hasBBox:     true,
		listedBBox:  [2]string{"POINT(0 0)", "POINT(3 3)"},
		closeRings:  true,
	},
	{
		description: "multipoint lacking data",
		twkbHex:     "0410",
		wkt:         "MULTIPOINT EMPTY",
	},
	{
		description: "multipoint no contents",
		twkbHex:     "040000",
		wkt:         "MULTIPOINT EMPTY",
		skipEncode:  true,
	},
	{
		description: "multipoint with bbox",
		twkbHex:     "04010408060803040604040404",
		wkt:         "MULTIPOINT(2 3,4 5,6 7)",
		postGIS:     "ARRAY ['POINT(2 3)'::geometry, 'POINT(4 5)'::geometry, 'POINT(6 7)'::geometry]",
		hasBBox:     true,
		listedBBox:  [2]string{"POINT(2 3)", "POINT(6 7)"},
	},
	{
		description: "multipoint z with bbox",
		twkbHex:     "040901040a030a0008020406080a0907",
		wkt:         "MULTIPOINT Z(2 3 4,7 -2 0)",
		hasZ:        true,
		postGIS:     "ARRAY ['POINT Z (2 3 4)'::geometry, 'POINT Z (7 -2 0)'::geometry]",
		hasBBox:     true,
		listedBBox:  [2]string{"POINT Z (2 -2 0)", "POINT Z (7 3 4)"},
	},
	{
		description: "multipoint m with bbox",
		twkbHex:     "040902040a030a0008020406080a0907",
		wkt:         "MULTIPOINT M(2 3 4,7 -2 0)",
		hasM:        true,
		postGIS:     "ARRAY ['POINT M (2 3 4)'::geometry, 'POINT M (7 -2 0)'::geometry]",
		hasBBox:     true,
		listedBBox:  [2]string{"POINT M (2 -2 0)", "POINT M (7 3 4)"},
	},
	{
		description: "multipoint z m with bbox",
		twkbHex:     "040903040a030a00080208020406080a0a090707",
		wkt:         "MULTIPOINT ZM(2 3 4 5,7 -2 0 1)",
		hasZ:        true,
		hasM:        true,
		postGIS:     "ARRAY ['POINT ZM (2 3 4 5)'::geometry, 'POINT ZM (7 -2 0 1)'::geometry]",
		hasBBox:     true,
		listedBBox:  [2]string{"POINT ZM (2 -2 0 1)", "POINT ZM (7 3 4 5)"},
	},
	{
		description: "multipoint z m with prec xy -1 & prec z 2 & prec m 3 & bbox",
		twkbHex:     "14096b040a030a00080208020406080a0a090707",
		wkt:         "MULTIPOINT ZM(20 30 0.04 0.005,70 -20 0 0.001)",
		hasZ:        true,
		hasM:        true,
		precXY:      -1,
		precZ:       2,
		precM:       3,
		postGIS:     "ARRAY ['POINT ZM (20 30 0.04 0.005)'::geometry, 'POINT ZM (70 -20 0 0.001)'::geometry]",
		hasBBox:     true,
		listedBBox:  [2]string{"POINT ZM (20 -20 0 0.001)", "POINT ZM (70 30 0.04 0.005)"},
	},
	{
		description: "multipoint with size & bbox & ids",
		twkbHex:     "04070b0004020402000200020404",
		wkt:         "MULTIPOINT(0 1,2 3)",
		postGIS:     "ARRAY ['POINT(0 1)'::geometry, 'POINT(2 3)'::geometry]",
		hasSize:     true,
		hasBBox:     true,
		listedBBox:  [2]string{"POINT(0 1)", "POINT(2 3)"},
		hasIDList:   true,
		listedIDs:   []int64{0, 1},
	},
	{
		description: "multilinestring lacking data",
		twkbHex:     "0510",
		wkt:         "MULTILINESTRING EMPTY",
	},
	{
		description: "multilinestring no contents",
		twkbHex:     "050000",
		wkt:         "MULTILINESTRING EMPTY",
		skipEncode:  true,
	},
	{
		description: "multilinestring",
		twkbHex:     "050002020000020203020202020202",
		wkt:         "MULTILINESTRING((0 0,1 1),(2 2,3 3,4 4))",
	},
	{
		description: "multipolygon lacking data",
		twkbHex:     "0610",
		wkt:         "MULTIPOLYGON EMPTY",
	},
	{
		description: "multipolygon no contents",
		twkbHex:     "060000",
		wkt:         "MULTIPOLYGON EMPTY",
		skipEncode:  true,
	},
	{
		description: "multipolygon with polygon lacking data",
		twkbHex:     "06000100",
		wkt:         "MULTIPOLYGON(EMPTY)",
		skipEncode:  true,
	},
	{
		description: "multipolygon with two polygons lacking data",
		twkbHex:     "0600020000",
		wkt:         "MULTIPOLYGON(EMPTY,EMPTY)",
		skipEncode:  true,
	},
	{
		description: "multipolygon unclosed rings with various contents",
		twkbHex:     "0600020001040000060000060500",
		wkt:         "MULTIPOLYGON(EMPTY,((0 0,3 0,3 3,0 3,0 0)))",
	},
	{
		description: "multipolygon unclosed rings",
		twkbHex:     "0600020104000006000006050001040802000202000001",
		wkt:         "MULTIPOLYGON(((0 0,3 0,3 3,0 3,0 0)),((4 4,4 5,5 5,5 4,4 4)))",
	},
	{
		description: "multipolygon closed rings",
		twkbHex:     "060002010500000600000605000005010508080002020000010100",
		wkt:         "MULTIPOLYGON(((0 0,3 0,3 3,0 3,0 0)),((4 4,4 5,5 5,5 4,4 4)))",
		closeRings:  true,
	},
	{
		description: "geometry collection lacking data",
		twkbHex:     "0710",
		wkt:         "GEOMETRYCOLLECTION EMPTY",
	},
	{
		description: "geometry collection no contents",
		twkbHex:     "070000",
		wkt:         "GEOMETRYCOLLECTION EMPTY",
		skipEncode:  true,
	},
	{
		description: "geometry collection with point and empty",
		twkbHex:     "070002010000020310",
		wkt:         "GEOMETRYCOLLECTION(POINT(0 1),POLYGON EMPTY)",
	},
	{
		description: "geometry collection",
		twkbHex:     "07000201000002020002080a0404",
		wkt:         "GEOMETRYCOLLECTION(POINT(0 1),LINESTRING(4 5,6 7))",
	},
	{
		description: "geometry collection with ids",
		twkbHex:     "070402000201000002020002080a0404",
		wkt:         "GEOMETRYCOLLECTION(POINT(0 1),LINESTRING(4 5,6 7))",
		postGIS:     "ARRAY ['POINT(0 1)'::geometry, 'LINESTRING(4 5,6 7)'::geometry]",
		hasIDList:   true,
		listedIDs:   []int64{0, 1},
	},
}

func TestTWKBUnmarshalMarshalValid(t *testing.T) {
	for _, tc := range TWKBTestCases {
		t.Run(tc.description, func(t *testing.T) {
			twkb := hexStringToBytes(t, tc.twkbHex)
			t.Logf("TWKB (hex): %v", tc.twkbHex)

			t.Run("decode", func(t *testing.T) {
				if tc.skipDecode {
					t.SkipNow()
				}

				t.Run("geometry", func(t *testing.T) {
					g, err := geom.UnmarshalTWKB(twkb)
					expectNoErr(t, err)
					expectGeomEqWKT(t, g, tc.wkt)
				})

				t.Run("envelope", func(t *testing.T) {
					gotExtEnv, ok, err := geom.UnmarshalTWKBEnvelope(twkb)
					expectNoErr(t, err)
					expectBoolEq(t, ok, tc.hasBBox)
					if ok {
						wantC1, ok := geomFromWKT(t, tc.listedBBox[0]).MustAsPoint().Coordinates()
						expectTrue(t, ok)
						wantC2, ok := geomFromWKT(t, tc.listedBBox[1]).MustAsPoint().Coordinates()
						expectTrue(t, ok)

						wantC1.X, wantC2.X = minMax(wantC1.X, wantC2.X)
						wantC1.Y, wantC2.Y = minMax(wantC1.Y, wantC2.Y)
						wantC1.Z, wantC2.Z = minMax(wantC1.Z, wantC2.Z)
						wantC1.M, wantC2.M = minMax(wantC1.M, wantC2.M)

						gotXY1, gotXY2, ok := gotExtEnv.XYEnvelope.MinMaxXYs()
						expectTrue(t, ok)

						ct := wantC1.Type
						expectCoordinatesTypeEq(t, ct, wantC2.Type)
						minZ, maxZ, hasZ := gotExtEnv.ZRange.MinMax()
						minM, maxM, hasM := gotExtEnv.MRange.MinMax()
						expectBoolEq(t, hasZ, ct.Is3D())
						expectBoolEq(t, hasM, ct.IsMeasured())

						expectXYEq(t, gotXY1, wantC1.XY)
						expectXYEq(t, gotXY2, wantC2.XY)

						if ct.Is3D() {
							expectFloat64Eq(t, minZ, wantC1.Z)
							expectFloat64Eq(t, maxZ, wantC2.Z)
						}
						if ct.IsMeasured() {
							expectFloat64Eq(t, minM, wantC1.M)
							expectFloat64Eq(t, maxM, wantC2.M)
						}
					}
				})

				t.Run("id list", func(t *testing.T) {
					got, has, err := geom.UnmarshalTWKBIDList(twkb)
					expectNoErr(t, err)
					expectBoolEq(t, has, tc.hasIDList)
					expectInt64SliceEq(t, got, tc.listedIDs)
				})

				t.Run("size", func(t *testing.T) {
					for _, extra := range []int{0, 13} {
						t.Run(fmt.Sprintf("append_%d", extra), func(t *testing.T) {
							buf := make([]byte, len(twkb)+extra)
							copy(buf, twkb)
							got, ok, err := geom.UnmarshalTWKBSize(buf)
							expectNoErr(t, err)
							expectBoolEq(t, ok, tc.hasSize)
							if tc.hasSize {
								expectIntEq(t, got, len(twkb))
							}
						})
					}
				})
			})

			t.Run("encode", func(t *testing.T) {
				if tc.skipEncode {
					t.SkipNow()
				}

				// Encode the WKT's geometry as TWKB and check its bytes match the expected TWKB bytes.
				g := geomFromWKT(t, tc.wkt)
				opts := []geom.TWKBWriterOption{}
				if tc.hasZ {
					opts = append(opts, geom.TWKBPrecisionZ(tc.precZ))
				}
				if tc.hasM {
					opts = append(opts, geom.TWKBPrecisionM(tc.precM))
				}
				if tc.hasSize {
					opts = append(opts, geom.TWKBSizeHeader())
				}
				if tc.hasBBox {
					opts = append(opts, geom.TWKBBoundingBoxHeader())
				}
				if tc.closeRings {
					opts = append(opts, geom.TWKBCloseRings())
				}
				if len(tc.listedIDs) > 0 {
					opts = append(opts, geom.TWKBIDList(tc.listedIDs))
				}
				marshaled, err := geom.MarshalTWKB(g, tc.precXY, opts...)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !bytes.Equal(twkb, marshaled) {
					t.Errorf("MarshalTWKB %x result differs from expected TWKB %x", marshaled, twkb)
				}
			})
		})
	}
}

// TestWriteTWKBSQLFile exists to allow testing of TWKB test cases
// against a PostGIS-enabled database.
//
// Note: some PostGIS implementations do not support all features,
// in particular some will always close polygon rings by duplicating
// the first point as the final point. So some test cases listed
// in this file may not agree with all PostGIS implementations.
func TestWriteTWKBSQLFile(t *testing.T) {
	t.Skip("this test was only used during initial development")
	sql := ""
	for _, tc := range TWKBTestCases {
		sql += "/* " + tc.description + " */\n"

		if !tc.skipDecode {
			sql += fmt.Sprintf("select ST_AsText(ST_GeomFromTWKB(E'\\\\x%s'));\n", tc.twkbHex)
		}

		if !tc.skipEncode {
			switch {
			case tc.hasIDList:
				ids := "ARRAY ["
				for _, id := range tc.listedIDs {
					ids += fmt.Sprintf("%d, ", id)
				}
				ids = ids[:len(ids)-2] + "]" // Remove trailing comma.

				sql += fmt.Sprintf("select ST_AsTWKB(%s, %s, %d, %d, %d, %v, %v);\n",
					tc.postGIS, ids, tc.precXY, tc.precZ, tc.precM, tc.hasSize, tc.hasBBox)
			case tc.precXY != 0 || tc.precZ != 0 || tc.precM != 0 || tc.hasSize || tc.hasBBox:
				sql += fmt.Sprintf("select ST_AsTWKB('%s'::geometry, %d, %d, %d, %v, %v);\n",
					tc.wkt, tc.precXY, tc.precZ, tc.precM, tc.hasSize, tc.hasBBox)
			default:
				sql += fmt.Sprintf("select ST_AsTWKB('%s'::geometry);\n", tc.wkt)
			}
		}
		sql += "\n\n"
	}

	err := os.WriteFile("../twkb_sql.txt", []byte(sql), 0o600)
	expectNoErr(t, err)
}

func TestZigZagInt(t *testing.T) {
	for _, tc := range []struct {
		n int64
		z uint64
	}{
		{0, 0},
		{-1, 1},
		{1, 2},
		{-2, 3},
		{2, 4},
		{-3, 5},
		{3, 6},
		{-4, 7},
		{4, 8},
		{-128, 255},
		{128, 256},
		{-32768, 65535},
		{32768, 65536},
	} {
		t.Run(strconv.Itoa(int(tc.n)), func(t *testing.T) {
			t.Logf("ZigZag encode int64: %v", tc.n)
			z := geom.EncodeZigZagInt64(tc.n)
			if tc.z != z {
				t.Fatalf("expected: %v, got: %v", tc.z, z)
			}
			t.Logf("ZigZag decode int64: %v", tc.z)
			n := geom.DecodeZigZagInt64(tc.z)
			if tc.n != n {
				t.Fatalf("expected: %v, got: %v", tc.n, n)
			}
		})
	}
}

func minMax(a, b float64) (float64, float64) {
	return math.Min(a, b), math.Max(a, b)
}
