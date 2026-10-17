# ntqrcode — Terminal QR codes for Bubble Tea

<p>
    <a href="https://pkg.go.dev/nimbleterminal.dev/ntqrcode/qrcode"><img src="https://pkg.go.dev/badge/nimbleterminal.dev/ntqrcode/qrcode.svg" alt="Go Reference"></a>
    <a href="https://github.com/NimbleTerminal/ntqrcode/actions/workflows/ci.yml"><img src="https://github.com/NimbleTerminal/ntqrcode/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="./CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg" alt="Code Of Conduct"></a>
</p>

Show QR codes in your [Bubble Tea](https://github.com/charmbracelet/bubbletea)
app, or generate them as images. `ntqrcode` uses Kitty graphics where
available and Unicode blocks everywhere else.

[go-qr](https://github.com/piglig/go-qr) does the QR generation. This library
adds a Bubble Tea component that fits codes to the available space, handles
resizing and graphics cleanup, and switches between Kitty images and Unicode
blocks. It also provides a small encoding API with input and image-size limits,
Unicode handling, and a quiet zone around every code.

Use go-qr directly for general QR generation, image export, or decoding. Use
`ntqrcode` when you want to display QR codes inside a Bubble Tea app.

Built for [gloss](https://github.com/NimbleMarkets/gloss), it's a small,
experimental companion to [NTCharts](https://github.com/NimbleMarkets/ntcharts).

[Open the browser demo](https://nimbleterminal.github.io/ntqrcode/)

```text
┌─────────────────────────────────────────┐
│                                         │
│                                         │
│    █▀▀▀▀▀█   ▄▀ █▄▄▀█▀▀▄ ▄▀▄ █▀▀▀▀▀█    │
│    █ ███ █ █▄ ▄█▀▄ █▀▀▀▄▀█▄▄ █ ███ █    │
│    █ ▀▀▀ █ █ ▀▀ ▀ █▄█▀ ▄█▄▀  █ ▀▀▀ █    │
│    ▀▀▀▀▀▀▀ █ ▀ █ █▄▀ █ ▀▄█▄█ ▀▀▀▀▀▀▀    │
│    █ ███▀▀▄ ▄█ ▄▀▀▀ ▀▄▄█▄▀▄▀ ██▀██▄▄    │
│     █▄ █ ▀  ▀▀██ ▄▀  ▀ ▄█▀▄█▀  █▄█▀▄    │
│    █  ▄█ ▀▄▄███▄▄▄█▄ ▀ ▄ ▀▄▀█▄ ▀█▄██    │
│    ▀▀▄▀▀▄▀▀█▀▀▄ ▄▄ ▄▀ █▄██▄████▀▄█▀     │
│    ▄█▀██▄▀▀ ▄█▀█▀▀ ▄█▄██▄▀▄▀▀▄█▀▄▄ █    │
│     ▄▄█▀█▀ ▄▀ █▀▄▀███▄ ███  ▄▄██▄██▄    │
│    ▄▀ ▀  ▀ ▀ ▄▄ ▀ █▀▄▄ ▀▄ █  ▄ ▀▀▄▀█    │
│    █ ▀  █▀██▄▄ ▄███▀ ▀ ▀█▄ ▄▀█▄ █▄▀     │
│    ▀     ▀▀▄▀▄ ███▀█▀ ▄▄ █ █▀▀▀█ ▄ ▄    │
│    █▀▀▀▀▀█ ▄▄█▀█▄▄█  ▀▄▀▄███ ▀ █▄█▄▄    │
│    █ ███ █ █▄▀█▄▀▄▄▄ ▀▀▄█ ▄██▀▀█▀ ██    │
│    █ ▀▀▀ █ ▀▄▄ ▄ ▄ ▄ ▄█▄▄ ▄ ▄█▀▄▄█      │
│    ▀▀▀▀▀▀▀ ▀ ▀▀  ▀  ▀ ▀  ▀▀▀▀ ▀   ▀     │
│                                         │
│                                         │
└─────────────────────────────────────────┘
```

## Quickstart

Requires Go 1.26.8+ and Bubble Tea v2. The API is experimental.
Development builds prefer Go 1.27.2 or newer; Go selects it automatically
with the default `GOTOOLCHAIN=auto` setting.

As of v0.2.0, the module is `nimbleterminal.dev/ntqrcode`. When upgrading
from v0.1.x, replace `github.com/NimbleMarkets/ntcharts-qrcode/qrcode` imports
with `nimbleterminal.dev/ntqrcode/qrcode`; the public API is unchanged.

```sh
go get nimbleterminal.dev/ntqrcode/qrcode
go run ./examples/make
```

The interactive generator accepts text or a URL. Press Enter to make a QR,
Ctrl+U to clear the input, Ctrl+G to switch graphics, and Ctrl+C to quit.
Editing the text clears the old code until you generate again. The browser
demo runs this same application, with all QR encoding performed locally.

`go run ./examples/qrcode` runs the original two-component example. It displays
two QR codes. Press `n` to change the first URL, `2` to
hide or restore the second code, `g` to switch rendering modes, and `q` to quit.
If your terminal is too narrow for both codes, press `2` to make room.

```go
import "nimbleterminal.dev/ntqrcode/qrcode"

code, err := qrcode.Encode("https://nimbleterminal.com/", qrcode.Options{})
if err != nil {
    return err
}
matrix := code.Matrix() // copy; true = black, includes the white quiet zone
img, err := code.Image(8) // opaque image.Image, exactly 8 pixels per module
// Handle err, then png.Encode(writer, img) to export it.
```

`Options{}` uses medium error correction, with an automatic boost when a
stronger level fits in the same size. You can also request `Low`, `Quartile`,
or `High`.

Set `MinVersion` and `MaxVersion` to limit the QR size (versions 1–40). Set
them to the same value to keep the size fixed as content changes. Text that
won't fit returns `ErrCapacity`. Unicode text is supported, including mixed
Japanese and Latin text.

## Composing the terminal component

```go
session := ntpicture.NewTerminalSession(ntpicture.TerminalOptions{})
qr, err := qrcode.New(code, qrcode.Config{
    Session: session, NextID: ntpicture.NextImageID,
})
// Handle err and retain both pointers. In the host's Init:
cmd := session.Init()

// In Update, update the session before reading its current state:
sessionCmd := session.Update(msg)
state := session.Snapshot()
cmd = tea.Batch(
    sessionCmd,
    qr.SetTerminal(preferKitty && state.Kitty == ntpicture.KittyCapabilitySupported,
        state.CellPixelWidth, state.CellPixelHeight),
    qr.SetSize(columns, rows),
    qr.Update(msg),
)
// Render qr.View() as a whole; inspect qr.Err() for failures.
// Before quitting, execute QR cleanup, then close the session:
cmd = tea.Sequence(qr.Close(), func() tea.Msg {
    session.Close()
    return tea.Quit()
})
```

**Execute every returned command.** See the
[complete host example](./examples/qrcode/main.go) for batching, native/WASM
startup, and cleanup before quitting.

- Share one host-owned `ntpicture.TerminalSession` across components on the
  same terminal. QR never initiates discovery or closes the host's session.
- `SetTerminal` trusts the host's boolean; pass true only after positive Kitty
  detection. It no longer consults process-global capability state. Pass false
  for unknown/unsupported capability or when the user selects glyphs.
- Broadcast messages to all components, including while showing glyphs or fit
  errors, so cancelled work can drain. Each filters its own presentation work.
- `NextID func() (int, error)` must allocate fresh positive 24-bit IDs shared
  with every picture in the application, including across replacement and resize.
  Pass `ntpicture.NextImageID` directly or a shared custom allocator. Allocation
  errors propagate through `Err()` and support `errors.Is`; do not reuse IDs.
  When upgrading an older `func() int` callback, return `(id, nil)` on success
  and an error when allocation fails.
- Execute commands from setters, `Update`, and `Close`. A model is a pointer
  with event-loop-owned state; do not copy it or mutate it concurrently.
- `SetCode(nil)` clears content. Encode replacements before setting them so
  invalid input can leave the old content intact. `Close` is idempotent and final.
- `Err()` also exposes asynchronous rendering and allocation failures.
- `errors.Is(qr.Err(), qrcode.ErrDoesNotFit)` identifies insufficient space.
  The component never truncates the matrix or removes the quiet zone.

`View()` returns a string you can include in your own `tea.View`. Add your
own keyboard bindings, borders, and labels around it.

## Rendering

Images use black modules on an opaque white background, with a four-module
quiet zone. They scale in whole pixels to keep the edges sharp. If a code
won't fit in the terminal, the component reports `ErrDoesNotFit` rather than
cutting it off.

Glyph rendering uses colored half-blocks and adapts to the terminal's cell
size. It starts with an 8×16-pixel estimate; pass measured dimensions through
`SetTerminal` when you have them.

**Apple Terminal** draws half-blocks with gaps that can break a QR code.
Use `Config{SolidCells: true}` to fill whole cells instead. This takes four
times the area—about 66×33 cells for a short URL. The example enables it when
`TERM_PROGRAM=Apple_Terminal`; you can also set `NTCHARTS_QRCODE_SOLID=1`,
including inside tmux.

Input must be nonempty UTF-8 and no more than 7,089 bytes. How much fits in a
code depends on the text and correction level. `Image(scale)` accepts integer
scales from 1 to 16, up to 2,048 pixels per edge. Terminal output is limited to
256 Kitty cells per axis or 65,536 glyph cells.

The original two-code example has been scanned with a phone in Apple Terminal (solid cells)
and iTerm2 (half-blocks). Kitty graphics and tmux have also been checked
visually. See [DEVELOP.md](./DEVELOP.md#manual-acceptance-checks) for the
recorded results and remaining checks.

## Development

```sh
task ci                 # full checks, builds, and vulnerability scans
task test               # tests without a vulnerability-database query
task build-ex-qrcode    # build the original two-code example
task build-ex-make      # build bin/ntqrcode-make
task run-make           # run the interactive generator
task serve-wasm-site    # http://localhost:8000/ntqrcode/
task clean              # remove generated binaries and demo assets
```

The browser demo site runs `examples/make` and `examples/qrcode` through
[go-booba](https://github.com/NimbleMarkets/go-booba). Its landing page follows
the sibling ntcanvas and ntpicture sites: `web/index.html`, `site.css`, and
`site.js` are tracked; the WASM binaries and runtime assets are generated.
`wasm.work` selects the Bubble Tea and clipboard forks needed for WASM;
native builds use upstream versions. The demos publish their keyboard controls
through booba actions, providing clickable Generate, Clear, and Graphics buttons
and controls for the two-code example.

Encoding uses [piglig/go-qr v2.6.0](https://github.com/piglig/go-qr/releases/tag/v2.6.0).
Rendering uses [ntpicture](https://github.com/NimbleTerminal/ntpicture).
The [migration guide](./DEVELOP.md#migrating-from-ntchartspicture-to-ntpicture)
records the dependency and API changes. We use the tagged ntpicture `v0.1.0`
release. Builds fetch it normally and require no sibling checkout; see the
[dependency verification steps](./DEVELOP.md#dependency-and-verification).

Tests independently decode generated images,
Kitty PNG data, and reconstructed glyph output. They also cover Unicode,
capacity limits, resizing, and cleanup. Camera and terminal checks are recorded
separately in [DEVELOP.md](./DEVELOP.md).

Run `task ci` before submitting changes. It checks formatting, dependencies,
race tests, vet, Windows/native/WASM builds, and known vulnerabilities. The
vulnerability scans require network access. The Pages workflow runs these
checks before deploying; configure GitHub Pages to use GitHub Actions to
enable it. See [DEVELOP.md](./DEVELOP.md) for implementation notes and the
dependency review.

## License

[MIT License](./LICENSE.txt) — Copyright (c) 2026 [Neomantra Corp](https://www.neomantra.com).
Dependency notices are in [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).

----
Made with :heart: and :fire: by the team behind [Nimble Terminal](https://nimbleterminal.com).
