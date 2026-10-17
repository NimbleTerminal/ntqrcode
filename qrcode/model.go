package qrcode

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	picture "nimbleterminal.dev/ntpicture"
)

var ErrDoesNotFit = errors.New("QR code doesn't fit")

// Config uses the host's image-ID allocator, shared with its other pictures.
// NextID must return a fresh, positive 24-bit ID on every call, including
// across components, or an error reported through Model.Err. Images are retired
// on replacement, resize, and Close. ntpicture.NextImageID can be passed directly.
//
// SolidCells makes the glyph fallback fill whole terminal cells with solid
// background color (about 2 columns × 1 row per module on 1:2 cells, four times
// the half-block area) instead of half-block glyphs. Hosts choose it for
// terminals such as Apple Terminal that draw U+2580 from the font, misplacing
// it within the cell. It does not affect Kitty rendering.
type Config struct {
	// Session supplies transport settings from the host-owned terminal connection.
	// The host runs its Init/Update/Close and passes capability and geometry via
	// SetTerminal. Nil retains the default session transport settings.
	Session    *picture.TerminalSession
	NextID     func() (int, error)
	SolidCells bool
}

// Model is a composable picture-style Bubble Tea component: setters and
// Update return commands for the host to run; View returns only its content.
// Do not copy a Model. The host must call Close and run its cleanup command
// before removing it. No Init/probe is needed: the host supplies capability
// and cell geometry through SetTerminal, from its terminal session.
type Model struct {
	code         *Code
	pic          *picture.Picture
	session      *picture.TerminalSession
	lastID       int
	w, h, cw, ch int
	kitty, solid bool
	glyph        string
	err          error
	closed       bool
}

func New(code *Code, cfg Config) (*Model, error) {
	if code == nil || len(code.modules) == 0 || cfg.NextID == nil {
		return nil, errors.New("QR component requires a code and the host's image-ID allocator")
	}
	m := &Model{code: code, session: cfg.Session, solid: cfg.SolidCells, cw: 8, ch: 16, err: ErrDoesNotFit}
	m.pic = picture.New(picture.Options{
		Session:        cfg.Session,
		ClearOnReplace: true,
		HostGeometry:   true,
		Render:         picture.Config{KittyZ: -1, Fit: picture.FitFill, KittyResolutionFactor: 1},
		NextID: func() (int, error) {
			id, err := cfg.NextID()
			if err != nil {
				return 0, err
			}
			if id <= 0 || id >= 1<<24 || id == m.lastID {
				return 0, errors.New("QR image allocator must return fresh positive 24-bit IDs")
			}
			m.lastID = id
			return id, nil
		},
	})
	return m, nil
}

// SetCode replaces content. Nil clears it. Encode errors can be handled before
// changing the displayed content. A failed fit never leaves the old QR visible.
func (m *Model) SetCode(code *Code) tea.Cmd {
	if m.closed || code == m.code {
		return nil
	}
	m.code = code
	return m.layout()
}

func (m *Model) SetSize(width, height int) tea.Cmd {
	width, height = max(0, width), max(0, height)
	if m.closed || (m.w == width && m.h == height) {
		return nil
	}
	m.w, m.h = width, height
	return m.layout()
}

// SetTerminal consumes the host's graphics choice and cell pixel dimensions.
// The host must pass true only after affirmative capability detection.
// This decision is authoritative; no process-global capability is consulted.
// The default geometry is 8x16 until the host supplies its measured size.
// Glyphs repeat whole columns and half-rows to approximate square modules
// (longer side at most 9/8 of the shorter). Kitty pixels remain exactly square.
func (m *Model) SetTerminal(kitty bool, cellWidth, cellHeight int) tea.Cmd {
	if m.closed || (m.kitty == kitty && m.cw == cellWidth && m.ch == cellHeight) {
		return nil
	}
	m.kitty, m.cw, m.ch = kitty, cellWidth, cellHeight
	return m.layout()
}

// Update accepts frames from the host event loop. Geometry is laid out before
// touching picture, so a cell-size reply cannot trigger a filtered QR resize.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.closed {
		return nil
	}
	switch v := msg.(type) {
	case uv.CellSizeEvent:
		// Explicit sessions are host-owned: their geometry arrives via SetTerminal.
		// Never let Picture resize the old raster before QR has rebuilt its modules.
		if m.session != nil || v.Width <= 0 || v.Height <= 0 {
			return nil
		}
		return m.SetTerminal(m.kitty, v.Width, v.Height)
	}
	// HostGeometry keeps terminal events out of Picture. Continue draining its
	// cancelled jobs even while showing custom glyphs or a layout error.
	return m.pic.Update(msg)
}

// Err reports layout, allocation, or asynchronous rendering failures.
// Insufficient space wraps ErrDoesNotFit. View never returns a partial QR.
func (m *Model) Err() error {
	if m.err != nil {
		return m.err
	}
	return m.pic.Status().Err
}

func (m *Model) View() string {
	if m.closed || m.code == nil {
		return ""
	}
	if err := m.Err(); err != nil {
		return err.Error()
	}
	if m.kitty {
		view := m.pic.View()
		if view == "" {
			return "Preparing QR…"
		}
		return view
	}
	return m.glyph
}

// Close clears pending frames and returns ntpicture's ID-specific delete
// command (including its tmux handling). It is idempotent.
func (m *Model) Close() tea.Cmd {
	m.closed, m.glyph = true, ""
	return m.pic.Close()
}

func (m *Model) layout() tea.Cmd {
	// Clear explicitly on failed fit, empty content, or a glyph transition.
	// Successful Kitty layouts replace image and geometry atomically in Submit.
	m.err, m.glyph = nil, ""
	if m.code == nil {
		return m.pic.SetImage(nil)
	}
	if len(m.code.modules) == 0 {
		m.err = errors.New("no QR code")
		return m.pic.SetImage(nil)
	}
	if m.cw < 1 || m.ch < 1 || m.cw > 256 || m.ch > 256 {
		m.err = errors.New("QR cell dimensions must be 1..256 pixels")
		return m.pic.SetImage(nil)
	}
	n := len(m.code.modules)
	if !m.kitty {
		x, y := glyphScale(m.cw, m.ch)
		if m.solid {
			x, y = solidScale(m.cw, m.ch)
		}
		cols, rows := n*x, (n*y+1)/2
		if cols > m.w || rows > m.h || cols*rows > 65536 {
			m.err = fmt.Errorf("%w: needs %d columns × %d rows in glyph mode", ErrDoesNotFit, cols, rows)
			return m.pic.SetImage(nil)
		}
		m.glyph = halfBlocks(m.code.modules, x, y, m.solid)
		return m.pic.SetImage(nil)
	}
	// Bound allocation, cell multiplication, and Kitty's diacritic grid.
	cols, rows := min(m.w, 256, MaxImageEdge/m.cw), min(m.h, 256, MaxImageEdge/m.ch)
	scale := min(8, cols*m.cw/n, rows*m.ch/n)
	if scale < 1 {
		m.err = fmt.Errorf("%w: needs at least %d × %d pixels including the quiet zone", ErrDoesNotFit, n, n)
		return m.pic.SetImage(nil)
	}
	src, err := m.code.Image(scale)
	if err != nil {
		m.err = err
		return m.pic.SetImage(nil)
	}
	side := n * scale
	cols, rows = (side+m.cw-1)/m.cw, (side+m.ch-1)/m.ch
	// FitFill + transparent Background + EXACT target-sized source bypasses
	// ntpicture's CatmullRom scaler. The source itself
	// is opaque white. ResolutionFactor must stay 1. Decoder tests inspect
	// the actual transmitted PNG, not only this source image.
	out := image.NewRGBA(image.Rect(0, 0, cols*m.cw, rows*m.ch))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(out, src.Bounds(), src, image.Point{}, draw.Src)
	// Each raster is newly allocated and never mutated after submission.
	return m.pic.Submit(picture.Frame{
		Image: out, Cols: cols, Rows: rows,
		Mode: picture.PictureKitty, CellWidth: m.cw, CellHeight: m.ch,
		Exact: true, Immutable: true,
	})
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Find the smallest whole-cell replication with nearly square modules.
// Exact ratios alone are impractical: 8x17 cells would require 17 columns
// and 16 half-rows per module. Allowing at most 12.5% aspect error keeps that
// case at one column/half-row, without filtering or losing any modules.
func glyphScale(cw, ch int) (int, int) {
	g := gcd(2*cw, ch)
	for x := 1; x < ch/g; x++ {
		y := max(1, (2*cw*x+ch/2)/ch)
		w, h := 2*cw*x, ch*y // doubled pixels to represent odd cell heights
		if 8*max(w, h) <= 9*min(w, h) {
			return x, y
		}
	}
	return ch / g, 2 * cw / g
}

// Find the smallest whole-cell module (columns, half-rows) that is nearly
// square, using the same 9/8 tolerance. Half-rows are always even, so each cell
// is a single solid color.
func solidScale(cw, ch int) (int, int) {
	g := gcd(cw, ch)
	for x := 1; x < ch/g; x++ {
		r := max(1, (2*cw*x+ch)/(2*ch))
		w, h := cw*x, ch*r
		if 8*max(w, h) <= 9*min(w, h) {
			return x, 2 * r
		}
	}
	return ch / g, 2 * cw / g
}

// halfBlocks maps modules directly to explicit black/white foreground AND
// background. The unused bottom half-row is white, never terminal-default.
// With solid, sy is even so foreground equals background in every cell and a
// space replaces the glyph, leaving nothing for a font to misplace.
func halfBlocks(matrix [][]bool, sx, sy int, solid bool) string {
	n := len(matrix)
	var out strings.Builder
	for y := 0; y < n*sy; y += 2 {
		if y > 0 {
			out.WriteByte('\n')
		}
		last := -1
		for x := range n * sx {
			top := matrix[y/sy][x/sx]
			bottom := y+1 < n*sy && matrix[(y+1)/sy][x/sx]
			fg, bg := 255, 255
			if top {
				fg = 0
			}
			if bottom {
				bg = 0
			}
			key := fg*256 + bg
			if key != last {
				fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm", fg, fg, fg, bg, bg, bg)
				last = key
			}
			if solid {
				out.WriteByte(' ')
			} else {
				out.WriteRune('▀')
			}
		}
		out.WriteString("\x1b[0m")
	}
	return out.String()
}
