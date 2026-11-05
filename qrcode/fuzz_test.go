package qrcode

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

func FuzzEncode(f *testing.F) {
	maxInt := int(^uint(0) >> 1)
	for _, payload := range []string{
		"", "\xff", "\x00\x1b]52;c;payload\a", "日本語¢£¬‖−〜🙂",
		strings.Repeat("1", MaxPayloadBytes), strings.Repeat("1", MaxPayloadBytes+1),
		strings.Repeat("a¢", 1000), "https://example.com/1234567890",
	} {
		f.Add(payload, uint8(0), 0, 0, 1)
	}
	f.Add("bounds", uint8(255), -maxInt-1, maxInt, maxInt)
	f.Add("version transition", uint8(1), 10, 26, 16)
	f.Add("version transition", uint8(2), 27, 40, -1)
	f.Fuzz(func(t *testing.T, payload string, level uint8, minV, maxV, scale int) {
		// Always exercise normal options too; random invalid options must not
		// starve payload segmentation and encoding of coverage.
		for _, opts := range []Options{{}, {Level: Level(level), MinVersion: minV, MaxVersion: maxV}} {
			code, err := EncodeContext(context.Background(), payload, opts)
			if err != nil {
				if code != nil {
					t.Fatal("returned code on failure")
				}
				continue
			}
			if len(payload) > MaxPayloadBytes || payload == "" || !utf8.ValidString(payload) || opts.Level > High {
				t.Fatal("accepted invalid input")
			}
			matrix := code.Matrix()
			n := len(matrix)
			if n < 29 || n > 185 || (n-25)%4 != 0 {
				t.Fatalf("invalid matrix size %d", n)
			}
			version := (n - 25) / 4
			if (opts.MinVersion != 0 && version < opts.MinVersion) || (opts.MaxVersion != 0 && version > opts.MaxVersion) {
				t.Fatal("ignored version bounds")
			}
			for y, row := range matrix {
				if len(row) != n {
					t.Fatal("non-square matrix")
				}
				for x, black := range row {
					if black && (x < QuietZone || y < QuietZone || x >= n-QuietZone || y >= n-QuietZone) {
						t.Fatal("quiet zone changed")
					}
				}
			}
			img, imageErr := code.Image(scale)
			validScale := scale >= 1 && scale <= MaxScale && n*scale <= MaxImageEdge
			if (imageErr == nil) != validScale {
				t.Fatal("image allocation bounds changed")
			}
			if imageErr == nil && (img.Bounds().Dx() != n*scale || img.Bounds().Dy() != n*scale) {
				t.Fatal("image geometry changed")
			}
			// An independent decoder checks actual generated pixels, including
			// control characters and the ambiguous Kanji segmentation boundary.
			img, err = code.Image(1)
			if err != nil {
				t.Fatal(err)
			}
			decode(t, img, payload)
		}
	})
}

func FuzzGeometry(f *testing.F) {
	maxInt := int(^uint(0) >> 1)
	for _, v := range [][4]int{{80, 40, 8, 16}, {0, 0, 0, 0}, {maxInt, maxInt, 256, 1}, {-maxInt - 1, maxInt, 1, 256}, {200, 200, 257, 16}, {200, 200, 8, 17}} {
		f.Add(v[0], v[1], v[2], v[3], false, false)
		f.Add(v[0], v[1], v[2], v[3], true, true)
	}
	f.Fuzz(func(t *testing.T, width, height, cw, ch int, kitty, solid bool) {
		m := component(t, "\x1b]52;c;payload\a", ids())
		m.solid = solid
		run(m.SetSize(width, height), m)
		run(m.SetTerminal(kitty, cw, ch), m)
		if cw < 1 || ch < 1 || cw > 256 || ch > 256 {
			if m.Err() == nil {
				t.Fatal("accepted invalid cell geometry")
			}
			return
		}
		if m.Err() != nil {
			if !errors.Is(m.Err(), ErrDoesNotFit) {
				t.Fatal(m.Err())
			}
			return
		}
		view := m.View()
		if len(view) > 4<<20 {
			t.Fatal("unbounded terminal output")
		}
		if !kitty {
			for _, r := range ansi.Strip(view) {
				if r != '▀' && r != ' ' && r != '\n' {
					t.Fatal("payload escaped into terminal output")
				}
			}
		}
	})
}
