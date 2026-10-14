// A small text-to-QR application shared by the native and browser demos.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	picture "nimbleterminal.dev/ntpicture"
	"nimbleterminal.dev/ntqrcode/internal/demorun"
	"nimbleterminal.dev/ntqrcode/qrcode"
)

const sample = "https://nimbleterminal.com/"

// Eight header rows, one spacer, and one help row surround the QR.
const chromeRows = 10

type model struct {
	terminal                              *picture.TerminalSession
	input                                 textinput.Model
	qr                                    *qrcode.Model
	width, height                         int
	preferKitty, solid, generated, closed bool
	err                                   error
	focus                                 tea.Cmd
}

func newModel() *model {
	return newModelWithSession(picture.NewTerminalSession(picture.TerminalOptions{}))
}

func newModelWithSession(session *picture.TerminalSession) *model {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "https://example.com or any text"
	// One extra character ensures over-limit input fails instead of silently
	// encoding a truncated payload. Encoding also enforces the UTF-8 byte limit.
	input.CharLimit = qrcode.MaxPayloadBytes + 1
	styles := input.Styles()
	styles.Cursor.Blink = false
	input.SetStyles(styles)
	input.SetValue(sample)
	input.CursorEnd()
	m := &model{
		terminal: session,
		input:    input, preferKitty: true, generated: true,
		solid: os.Getenv("TERM_PROGRAM") == "Apple_Terminal" || os.Getenv("NTCHARTS_QRCODE_SOLID") == "1",
	}
	m.focus = m.input.Focus()
	code, err := qrcode.Encode(sample, qrcode.Options{})
	if err != nil {
		panic(err)
	}
	m.qr, err = qrcode.New(code, qrcode.Config{Session: m.terminal, NextID: nextID, SolidCells: m.solid})
	if err != nil {
		panic(err)
	}
	return m
}

// Bridge the shared, non-recycling allocator to the QR component's older API.
// Zero lets the component report exhaustion through Err without reusing an ID.
func nextID() int {
	id, err := picture.NextImageID()
	if err != nil {
		return 0
	}
	return id
}

func (m *model) Init() tea.Cmd { return tea.Batch(m.terminal.Init(), m.focus) }

func (m *model) generate() tea.Cmd {
	code, err := qrcode.Encode(m.input.Value(), qrcode.Options{})
	m.err = err
	m.generated = err == nil
	if err != nil {
		return m.qr.SetCode(nil)
	}
	return m.qr.SetCode(code)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.closed {
		return m, nil
	}
	cmds := []tea.Cmd{m.terminal.Update(msg)}
	before := m.input.Value()
	handled := false
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(0, msg.Width), max(0, msg.Height)
		m.input.SetWidth(max(1, m.width-3)) // prompt plus cursor
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.closed = true
			return m, tea.Sequence(m.qr.Close(), func() tea.Msg { m.terminal.Close(); return tea.Quit() })
		case "enter":
			cmds = append(cmds, m.generate())
			handled = true
		case "ctrl+g":
			m.preferKitty = !m.preferKitty
			handled = true
		case "ctrl+u":
			m.input.Reset()
			handled = true
		}
	}
	if !handled {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.input.Value() != before {
		// Never display a code for old text next to a newly edited value.
		m.generated, m.err = false, nil
		cmds = append(cmds, m.qr.SetCode(nil))
	}
	state := m.terminal.Snapshot()
	cw, ch := state.CellPixelWidth, state.CellPixelHeight
	cmds = append(cmds, m.qr.SetTerminal(m.preferKitty && state.Kitty == picture.KittyCapabilitySupported, cw, ch))
	w, h := m.width, max(0, m.height-chromeRows)
	if m.width < 24 || m.height < 14 {
		w, h = 0, 0
	}
	cmds = append(cmds, m.qr.SetSize(w, h), m.qr.Update(msg))
	return m, tea.Batch(cmds...)
}

func (m *model) View() tea.View {
	if m.closed {
		return tea.NewView("")
	}
	if m.width < 24 || m.height < 14 {
		v := tea.NewView(ansi.Truncate("Enlarge to at least 24 × 14", m.width, "…"))
		v.AltScreen = true
		return v
	}
	clip := func(s string) string { return ansi.Truncate(s, m.width, "…") }
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	mode := "glyph"
	if m.solid {
		mode = "solid cells"
	}
	if m.preferKitty && m.terminal.Snapshot().Kitty == picture.KittyCapabilitySupported {
		mode = "Kitty"
	}
	status := "Press Enter to generate."
	content := "Your QR code will appear here."
	switch {
	case m.err != nil:
		status = "Cannot generate: " + m.err.Error()
	case m.generated:
		status = "Ready to scan · " + mode
		content = m.qr.View()
		if m.qr.Err() != nil {
			status = "Cannot display QR: " + m.qr.Err().Error()
			if errors.Is(m.qr.Err(), qrcode.ErrDoesNotFit) {
				status = "More room needed: enlarge the terminal or shorten the text."
			}
			content = clip(m.qr.Err().Error())
		} else if content == "Preparing QR…" {
			status = "Preparing QR…"
		}
	}
	header := []string{
		accent.Render("Make a QR code"),
		clip("Type or paste text, then press Enter."),
		"",
		clip(fmt.Sprintf("Text or URL · %d / %d bytes", len(m.input.Value()), qrcode.MaxPayloadBytes)),
		m.input.View(),
		"",
		clip(status),
		"",
	}
	body := lipgloss.NewStyle().Height(m.height - chromeRows).Render(content)
	hint := clip("Enter make · Ctrl+U clear · Ctrl+G graphics · Ctrl+C quit")
	v := tea.NewView(strings.Join(header, "\n") + "\n" + body + "\n\n" + hint)
	v.AltScreen = true
	return v
}

func main() {
	if err := demorun.Run(newModel(), "make", "Make a QR code", demorun.Keys{
		Primary: []key.Binding{demorun.Binding("enter", "Generate"), demorun.Binding("ctrl+u", "Clear"), demorun.Binding("ctrl+g", "Graphics")},
		More:    []key.Binding{demorun.Binding("ctrl+c", "Quit")},
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
