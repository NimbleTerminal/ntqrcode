package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	picture "nimbleterminal.dev/ntpicture"
	"nimbleterminal.dev/ntqrcode/qrcode"
)

// Drain actual host commands, including ordered transmissions and cleanup.
func pump(m *model, cmd tea.Cmd) (raw string) {
	var visit func(tea.Cmd)
	visit = func(c tea.Cmd) {
		if c == nil {
			return
		}
		msg := c()
		if msg == nil {
			return
		}
		if r, ok := msg.(tea.RawMsg); ok {
			raw += fmt.Sprint(r.Msg)
			return
		}
		if _, ok := msg.(tea.QuitMsg); ok {
			return
		}
		v := reflect.ValueOf(msg)
		if v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i).Interface().(tea.Cmd))
			}
			return
		}
		_, next := m.Update(msg)
		visit(next)
	}
	visit(cmd)
	return
}

func send(m *model, msg tea.Msg) string {
	_, cmd := m.Update(msg)
	return pump(m, cmd)
}

func glyphModel(t *testing.T) *model {
	t.Helper()
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("NTCHARTS_QRCODE_SOLID", "0")
	picture.ForceKittyCapability(picture.KittyCapabilityUnsupported)
	t.Cleanup(func() { picture.ForceKittyCapability(picture.KittyCapabilityUnknown) })
	m := newModelWithSession(testSession(picture.KittyCapabilityUnsupported))
	send(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	return m
}

func TestMakeEditsAndGeneratesExactUnicodePayload(t *testing.T) {
	m := glyphModel(t)
	send(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	text := "https://example.org/東京?q=☕&g=2"
	send(m, tea.PasteMsg{Content: text})
	if m.input.Value() != text || m.qr.View() != "" || m.generated {
		t.Fatal("editing must retain the text and clear the previous code")
	}
	send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.err != nil || !m.generated || m.qr.Err() != nil {
		t.Fatalf("generation failed: %v / %v", m.err, m.qr.Err())
	}
	code, err := qrcode.Encode(text, qrcode.Options{})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := qrcode.New(code, qrcode.Config{NextID: nextID})
	if err != nil {
		t.Fatal(err)
	}
	expected.SetSize(100, 40)
	if m.qr.View() != expected.View() {
		t.Fatal("QR does not encode the entered payload")
	}
	send(m, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if m.closed || !strings.HasSuffix(m.input.Value(), "q") || m.qr.View() != "" {
		t.Fatal("q must edit, not quit")
	}
}

func TestInvalidInputClearsCodeAndRecovers(t *testing.T) {
	m := glyphModel(t)
	for _, text := range []string{"", strings.Repeat("a", 4000), strings.Repeat("é", 4000), strings.Repeat("x", 100000)} {
		send(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
		send(m, tea.PasteMsg{Content: text})
		send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.err == nil || m.generated || m.qr.View() != "" {
			t.Fatal("invalid input left a QR visible")
		}
		if len([]rune(m.input.Value())) > qrcode.MaxPayloadBytes+1 {
			t.Fatal("unbounded input")
		}
	}
	send(m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	send(m, tea.PasteMsg{Content: "hello"})
	send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.err != nil || !m.generated || m.qr.View() == "" {
		t.Fatal("could not recover from error")
	}
}

func TestLayoutKeepsWholeQRAndFitsViewport(t *testing.T) {
	m := glyphModel(t)
	for _, solid := range []bool{false, true} {
		m.qr.Close()
		code, _ := qrcode.Encode(sample, qrcode.Options{})
		m.qr, _ = qrcode.New(code, qrcode.Config{NextID: nextID, SolidCells: solid})
		for _, h := range []int{14, 24, 50} {
			for w := 24; w <= 100; w++ {
				send(m, tea.WindowSizeMsg{Width: w, Height: h})
				view := m.View().Content
				if ansi.StringWidth(strings.Split(view, "\n")[0]) > w || len(strings.Split(view, "\n")) > h {
					t.Fatalf("view overflows %dx%d", w, h)
				}
				for _, row := range strings.Split(view, "\n") {
					if ansi.StringWidth(row) > w {
						t.Fatalf("row overflows width %d: %q", w, row)
					}
				}
				if m.qr.Err() == nil {
					for _, row := range strings.Split(m.qr.View(), "\n") {
						if !strings.Contains(view, row) {
							t.Fatalf("QR cropped at %dx%d", w, h)
						}
					}
				}
			}
		}
	}
}

func TestKittyCleanupOnEditAndQuit(t *testing.T) {
	m := newModelWithSession(testSession(picture.KittyCapabilitySupported))
	send(m, tea.WindowSizeMsg{Width: 100, Height: 50})
	raw := send(m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	if !strings.Contains(raw, "a=d,d=I,i=") || m.qr.View() != "" {
		t.Fatal("edit did not retire Kitty QR")
	}
	send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	raw = send(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !strings.Contains(raw, "a=d,d=I,i=") || !m.closed || m.qr.View() != "" {
		t.Fatal("quit did not retire Kitty QR")
	}
}

func testSession(capability picture.KittyCapability) *picture.TerminalSession {
	tmux := false
	return picture.NewTerminalSession(picture.TerminalOptions{Kitty: capability, SharedMemory: picture.KittyCapabilityUnsupported, TmuxPassthrough: &tmux})
}
