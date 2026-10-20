# Developing ntqrcode

Run `task ci` before submitting changes. The module follows the sibling
NTCharts projects: public component in `qrcode/`, a runnable example in
`examples/qrcode/`, an interactive generator in `examples/make/`, native Task
builds, and a booba WASM demo.

The component was extracted from gloss; application overlays, key bindings,
URL selection, and export-file policy remain in gloss.

## Component and dependency contract

`qrcode.Encode(text, Options{Level: Medium})` returns an immutable
`Code`. `Matrix()` returns a copy including four white quiet-zone modules on
every side; `Image(scale)` returns an opaque black/white `image.Image` for
`png.Encode`. Medium (M) is the default minimum; Low (L), Quartile (Q), and
High (H) are supported, and the encoder raises the level for free when the
payload still fits the chosen symbol version. Numeric, alphanumeric, byte,
and Kanji runs are optimized; non-ASCII payloads carry a UTF-8 ECI header.
Characters with ambiguous Kanji mappings stay in UTF-8 byte segments.
`Options.MinVersion` and
`MaxVersion` bound the symbol version (1..40); setting both pins an exact
size. Empty/invalid UTF-8 and invalid options are errors. Input is
bounded to 7,089 bytes before UTF-8 validation or encoding; actual capacity
depends on content and correction. Capacity errors wrap `ErrCapacity`. Image scale is an integer
1..16 and the edge cannot exceed 2,048 pixels.

`New(code, Config{NextID: allocator})` constructs the picture-style component.
The allocator must return fresh positive 24-bit IDs shared with the host's other
pictures; gloss passes `nextKittyID`. Drive `SetCode`, `SetSize`, `SetTerminal`,
and `Update` from the existing Bubble Tea loop and execute their returned
commands. Render `View()` as a whole. `Err()` wraps `ErrDoesNotFit` when there
is insufficient room; no partial QR is rendered. Before removal call `Close()`
and execute its cleanup command. It is idempotent and rejects late frames.

There is no terminal Init, input read, mode change, or synchronous probe.
The host's TerminalSession performs asynchronous capability detection; the QR
component receives the confirmed choice and measured cell size through
SetTerminal. Pass false for unknown/unsupported Kitty capability to fall back to explicit black/white Unicode half-blocks built directly
from modules. Glyphs use the smallest integer column/half-row replication
whose longer module side is at most 9/8 of its shorter side. This slight
aspect tolerance avoids enormous layouts for ordinary sizes such as 8×17;
glyph modules are approximately square, with sharp edges and no resampling.
The initial 8×16 cell estimate assumes a 1:2 font. Unusual measured ratios
can still require more space. Kitty and PNG modules remain exactly square.

Apple Terminal draws U+2580 from the font: measured in a 30px cell, the upper
half block landed 5px low and 13px tall, so finder and alignment patterns
broke while background fills stayed exact. `Config.SolidCells` therefore fills
whole cells with identical foreground/background and writes spaces, using the
same 9/8 tolerance (2×1 on 1:2 cells) and four times the half-block area. The
host chooses it; the library does not detect terminals. Tests require equal
colors in every cell and no `▀`, and decode the physical result.

For Kitty, the component builds an opaque bitmap with square, integer-sized
modules (up to eight pixels each), padded white to exactly fill its cell
rectangle. ntpicture normally uses Catmull–Rom resizing; its exact-size `FitFill`
path bypasses resampling. We retain a transparent configured background and
an exact source/target size. The QR source itself remains opaque. Resolution stays
1.0. Tests inspect and independently decode the transmitted PNG, guarding this
dependency behavior. Until the frame is transmitted, the component shows a
preparation message while Picture has no committed view. QR uses Picture only
for Kitty output; glyphs remain direct module rendering. Content/layout changes
clear failed fits and replace valid rasters atomically with fresh IDs. Picture owns ordered
transmission, coalescing, cancellation, tmux handling, and late cleanup. Presentation allocations are
bounded to 2,048-pixel edges, 256 Kitty cells per axis, or 65,536 glyph cells.

Encoder review (October 2026):

- [piglig/go-qr](https://github.com/piglig/go-qr) v2.7.0 is the encoder:
  MIT-licensed pure Go with no external runtime dependencies, implementing QR
  Model 2 versions 1–40, four correction levels, optimized mixed-mode
  segmentation, mask selection, and Reed–Solomon correction. It is a young,
  single-author project; we accept that risk because the decode-based tests
  below validate every symbol independently, and a differential corpus against
  skip2/go-qrcode (decode round-trips plus symbol sizes, 176 cases) matched.
  We use only its segment/matrix API and own the quiet zone, bounded image,
  and terminal presentation layers.
- The encoder is pinned to the upstream v2.7.0 release, which includes the
  version-search fix from [PR #100](https://github.com/piglig/go-qr/pull/100).
  No encoder fork or `replace` directive is needed. V2 makes its optimizer
  private, so `qrcode/segments_optimal.go` adapts its MIT-licensed dynamic
  program for safe runs using public segment constructors. We retain at most
  three segmentation passes across character-count ranges (1–9, 10–26,
  27–40), then perform a bounded `EncodeSegments` search that includes our
  UTF-8 ECI header and respects caller-supplied version bounds. A differential
  test compares safe-payload versions and correction levels with upstream v2.
- Its Kanji table has seven Unicode mappings that differ from common
  Shift-JIS decoders: backslash, `¢`, `£`, `¬`, `‖`, `−`, and `〜`. We split
  around those characters and encode them in byte mode, preserving optimized
  runs on either side. A whole-payload byte alternative prevents splitting
  overhead from inflating the symbol or rejecting byte-encodable text.
  When both candidates use the same version, we choose the strongest
  correction either can fit, keeping the version fixed. Tests independently
  decode both the payload and the correction level in each selection direction.
  Non-ASCII payloads include UTF-8 ECI assignment 26, counted toward capacity.
  Regression tests decode individual characters and mixed Japanese/text
  payloads, exercise numeric compression and byte fallback, and check the
  version transitions and ECI overhead.
- [skip2/go-qrcode](https://github.com/skip2/go-qrcode), the first encoder, is
  unmaintained since 2024 and lacks optimal segmentation and correction
  boosting. Its `Bitmap()` included the quiet zone; the wrapper now adds it.
- [yeqown/go-qrcode](https://github.com/yeqown/go-qrcode) was also considered:
  it has recent maintenance (August 2026); its v2 module has external
  Reed–Solomon and x/text dependencies and a broader
  customization surface. Style and logos are out of scope.
- [gozxing](https://github.com/makiuchi-d/gozxing) v0.1.1 is the independent,
  MIT-licensed pure-Go decoder used only in tests. It is not linked into gloss.
  Tests cover Unicode, all correction levels, version bounds, capacity, quiet
  zones, integer scaling, reconstructed colored half-blocks, actual Kitty PNG
  bytes, fallback, resize, content replacement, separate instances, and
  cleanup. Dense one-pixel images use ZXing's pure-symbol extraction path;
  ordinary finder-pattern detection is also tested on a default-size URL image.

## Automation and companion conventions

The library and demos are separate Go modules. `examples/go.mod` owns go-booba,
its `booba-assets` tool, Bubbles textinput, and Lip Gloss, with a local replacement
for the root ntqrcode module. Run demos with `go -C examples run ./make` or
`go -C examples run ./qrcode`. Native Task commands set `GOWORK=off` and test,
vet, tidy, verify, and build both modules explicitly. `wasm.work` includes both
modules and applies only the browser forks. Notices use both demo dependency
graphs; vulnerability scans cover the library and demos in native and WASM builds.
The downstream consumer regression also checks that go-booba stays out of the
library’s module graph.

GitHub Actions uses `actions/checkout@v7`, `actions/setup-go@v7`, and
`go-task/setup-task@v2`, following the main NTCharts repo. CI invokes `task ci`
so local and hosted checks stay in sync. Hosted checks use Go 1.27.2.
`go.mod`, `examples/go.mod`, and `wasm.work` prefer Go 1.27.2 or newer, so
plain `task ci` selects a patched toolchain with the default `GOTOOLCHAIN=auto` setting. Explicit
`GOTOOLCHAIN` overrides take precedence; use `GOTOOLCHAIN=go1.27.2 task ci`
to match hosted checks exactly. `TestDownstreamConsumer` creates a
separate main module with `GOWORK=off` and only a local replacement for this
library; it verifies dense encoding and capacity rejection without inheriting
any of the library's dependency replacements.

`task ci` includes `task vuln`, which installs `govulncheck` v1.8.0 into the
ignored `.task/bin/` directory. The scanner runs as a host executable, first
against native packages and tests with `GOWORK=off`, then against js/wasm
packages using `wasm.work` so the WASM forks are covered. The scanner version
is pinned in Taskfile.yml; the vulnerability database stays current. Both
reachable vulnerability findings and scanner/network failures stop CI and
therefore prevent Pages deployment. Run `task vuln` on its own for a security
check; `task test` remains available without the vulnerability-database query.

Like the SVG/PDF companions, weekly Dependabot checks cover Actions and Go
modules. Task targets include `go-update`, `clean`, and `clean-wasm-site`;
`serve-wasm-site` serves `/ntqrcode/` to match the GitHub Pages prefix.
The site shell (`web/index.html`, `site.css`, `site.js`) is tracked and
follows the ntcanvas/ntpicture layout: demo cards select `?demo=make` or
`?demo=qrcode`, `site.js` fetches `demos/<name>.wasm`, installs the booba
action bar and renderer HUD, and reports status below the terminal frame.
`build-wasm-site` copies `LICENSE.txt` and `THIRD_PARTY_NOTICES.md` next to
the page for the footer links; booba-assets never overwrites the tracked page.
Pages builds on `main` pushes or manual dispatch, with deployment permissions
limited to the deploy job. Its build job runs the full `task ci` checks on
the selected revision before uploading the site, so automatic and manual
deployments require passing checks. Repository Pages settings must use
GitHub Actions.

## Releasing

The module path is `nimbleterminal.dev/ntqrcode`; source and tags live
at `https://github.com/NimbleTerminal/ntqrcode`. Before publishing, ensure
`https://nimbleterminal.dev/ntqrcode?go-get=1` (and package subpaths) serves
HTML containing this metadata instead of redirecting to a marketing site:

```html
<meta name="go-import" content="nimbleterminal.dev/ntqrcode git https://github.com/NimbleTerminal/ntqrcode">
```

The `v0.3.0` release changes `Config.NextID` from `func() int` to
`func() (int, error)`, and `SetTerminal(true, ...)` now trusts host capability
detection. Keep both migration notes visible in the release description. The
root module retracts `v0.1.0`–`v0.1.1`, whose tags declare the former module path;
the retractions become discoverable when this version is published. Keep
those Git tags intact. The examples module is for checkout-local demos and does
not need its own release tag.

Update the dated changelog and dependency notices, run
`GOTOOLCHAIN=go1.27.2 task ci`, and commit the release changes. Publish the
version tag and GitHub release only after module discovery works. Then verify
`go get nimbleterminal.dev/ntqrcode/qrcode@v0.3.0` from a fresh module without
local replacements. The downstream consumer test uses a local replacement,
so it does not verify the public vanity endpoint or published tag.

## Manual acceptance checks

Run the example in a Kitty-capable terminal and a glyph-only terminal, then
through tmux with its normal graphics setup. Resize and change fonts, switch
render modes, change content, remove/restore the second code, and quit. Check
for stale images and scan both codes with a phone. Record terminal, font, cell
geometry, transport, and exact decoded URL. Automated decoders are not a
substitute.

Results, 2026-10-06 (Evan Wies; detail recorded only as reported):

- Apple Terminal, glyph fallback: half-block `▀` is drawn from the font and
  misplaced (about 5px low, 13px tall in a 30px cell; other block glyphs also
  appeared inset from the cell edges), breaking finder and alignment patterns. With
  `Config.SolidCells` the codes render correctly at 70×40 or larger and scan
  with a phone.
- iTerm2, glyph fallback with half-blocks: renders correctly and scans.
- A Kitty-graphics terminal and tmux with its normal graphics setup: checked
  and reported fine. Phone scans of these were not reported.
- Not recorded: the Kitty terminal's identity, fonts, exact cell geometry, and
  the decoded URLs. WezTerm and other glyph terminals were not tried. Add
  those details here if they are re-run.

Results, 2026-10-17 (user report and supplied screenshot):

- Apple Terminal, generator with `NTCHARTS_QRCODE_SOLID=1`: the user reported
  that the displayed code would not scan with a phone. Font, cell geometry,
  terminal version, and phone/scanner model were not supplied.
- Screenshot measurement found modules 28 × 30 pixels (about 7% taller than
  square), with intact white padding. Independent automated decoding of the
  screenshot returned exactly `https://nimbleterminal.com/`.
- This is not a successful phone scan. Camera acceptance for this configuration
  remains unresolved; reducing the displayed size or increasing camera distance
  was suggested but no retry result has been reported.

## Migrating from NTCharts/picture to ntpicture

The first migration changed imports and used ntpicture's compatible low-level
`Model` (`NewWithConfig` became `NewModelWithConfig`). The completed lifecycle
migration uses one long-lived `*ntpicture.Picture` per QR component instead.
Both example hosts now use `*ntpicture.TerminalSession` for discovery.

1. Replace the NTCharts picture import with
   `picture "nimbleterminal.dev/ntpicture"`. Construct the owned renderer with
   `picture.New(picture.Options{ClearOnReplace: true, HostGeometry: true, ...})`.
   Do not recreate it on each layout: it must drain cancelled jobs before scheduling newer work.
2. Keep QR encoding, fit validation, integer module rasterization, white padding,
   and direct half-block/solid-cell glyph rendering in this package. Configure
   the Picture with `FitFill`, transparent render background, and resolution 1.
   Submit the exact cell-sized opaque raster via `Submit(Frame{Image: raster,
   Cols: cols, Rows: rows, Mode: PictureKitty, CellWidth: cw, CellHeight: ch,
   Exact: true, Immutable: true})`. Image and geometry change as one generation;
   Exact rejects mismatched raster dimensions. Each submission owns newly
   allocated pixels that are never subsequently mutated.
3. Clear with `SetImage(nil)` on a failed QR fit, empty content, or transition
   to custom glyphs. Valid Kitty layouts use Submit directly: ClearOnReplace
   clears their previous front, including on a rejected submission. Do not
   prepend a separate clear or call geometry setters on the old source.
   While no front is committed, show the host-facing preparation message.
4. Delete local frame-ready messages, accepted-output guards, and transmission
   wrapping. Picture owns those races. Forward internal messages through
   `Picture.Update`, even while QR shows glyphs or a fit error. Run every command
   from clearing, Submit, Update, and Close.
   Expose `Picture.Status().Err` through QR's `Err()` alongside layout errors.
5. Use `Config.NextID func() (int, error)`, matching Picture’s allocator.
   Pass `ntpicture.NextImageID` directly or share a custom allocator across
   components. This replaces the earlier `func() int` API: old callbacks must
   return `(id, nil)` on success and an error on failure. Preserve allocator
   errors through `Err()` and reject invalid or immediately repeated IDs.
   The host remains responsible for uniqueness across all components.
   Glyph rendering does not allocate image IDs. Fresh IDs and immediate clearing
   are required here; stable-ID streaming is not appropriate for QR replacement.
6. Add optional `Config.Session` for transport settings. The host owns session
   Init/Update/Close and calls QR's `SetTerminal` with confirmed capability and
   measured geometry. **Behavior change:** true is now authoritative and no
   longer gated on process-global detection. Existing hosts must combine the
   graphics preference with affirmative capability detection themselves.
7. Set HostGeometry so Picture.Update ignores all terminal events, including
   private probe ticks. With an explicit session, let the host update that
   session, read Snapshot, then call QR's SetTerminal before QR.Update. Without
   a session, QR handles valid cell reports itself and rebuilds the raster first.
   Other events can be forwarded without QR maintaining a terminal-message
   filter. Discovery responses belong to the host. Close all QR components and
   run their commands before closing the shared session; removing one QR must not close the session.
8. Tests should inspect emitted `tea.RawMsg` Kitty APC bytes, not expect public
   `KittyFrameMsg` preparation messages: Picture's preparation messages are
   private. Retain PNG pixel equality and independent decoding, reconstructed
   glyph decoding, multiple owners, coalescing, failed-fit clearing, allocator
   errors/recovery, session geometry, and cleanup after accepted output even
   when the component has been removed from message routing.
9. Tidy dependencies, regenerate notices, run `task test`, native examples, and
   `task ci`. Preserve WASM-only forks. Pin a publicly fetchable tag or commit;
   the tagged dependency is described below.

Automated decoding and builds do not establish phone scanning or visual parity.
Native terminals, fonts, tmux, browser renderers, and camera scans still require
manual acceptance and a recorded result.

### Dependency and verification

Development pins `nimbleterminal.dev/ntpicture v0.1.1`.
The root module has no local replacement; native and WASM builds resolve the
same ntpicture revision without requiring a sibling checkout.

The downstream smoke test creates a separate main module with a local
replacement for ntqrcode only. It fetches ntpicture through normal Go module
resolution, checking that consumers can use this dependency pin too.

To upgrade to a later public commit or release, substitute its full commit hash
or tag for `REVISION`:

```sh
go get nimbleterminal.dev/ntpicture@REVISION
go mod tidy
task notices
task ci
```

Commit go.mod/go.sum, workspace checksums, and generated notices together.
Keep the actual-Kitty-PNG and reconstructed-glyph decoding tests when upgrading.

Public dependency verification, 2026-10-16: `task ci` passed against commit
`e6f89253dcf5` with no dependency replacement or consumer override.
This includes downstream resolution, exact Kitty PNG and glyph decoding,
race tests, native/Windows/WASM builds, browser action/interaction smoke checks,
and both vulnerability scans (no findings). Tests now use current session
configuration instead of removed legacy global capability/transport setters.
No additional manual visual or camera checks were performed.

Allocator and dependency update, 2026-10-17: pinned public commit `30268e212552`
and changed `Config.NextID` to `func() (int, error)`. Both examples pass
`ntpicture.NextImageID` directly. Full `task ci` passed, including downstream
compilation with that allocator, allocator error identity/recovery, exact PNG
and glyph decoding, browser action/interaction smoke checks, and native/WASM
vulnerability scans (no findings). No manual visual or camera checks were added.

Tagged release verification, 2026-10-17: upgraded to ntpicture `v0.1.0`.
Full `task ci` passed, including downstream resolution, exact Kitty PNG and glyph
decoding, native/Windows/WASM builds, browser smoke checks, and both vulnerability
scans (no findings). No additional manual visual or camera checks were performed.

The following historical checks used the temporary `../ntpicture` replacement
and an explicit `NTQRCODE_TEST_NTPICTURE` consumer override. That override has
now been removed so the consumer test always exercises normal dependency
resolution.

Initial compatibility-migration verification: `task test`, `task build-ex-qrcode`, and the full
`task ci` gate passed with the then-active explicit local consumer override. This
includes race tests, vet, module tidy/verification, Windows/native/WASM builds,
generated notices, and native/WASM vulnerability scans (no vulnerabilities
reported). The decoding tests exercised actual Kitty PNG payloads and
reconstructed glyph output. No new terminal, browser visual, or camera checks
were performed for this migration.

Revalidated against sibling HEAD `b12e0a8` and its current working tree after
the raster geometry, bounded cover/glyph allocation, and playback recovery
updates: the full CI gate passed with the local consumer override, and the
native demo was force-rebuilt to include sibling changes. Both vulnerability
scans again reported no vulnerabilities. The compatibility migration needed no
additional QR source changes.


Picture lifecycle migration verification, 2026-10-14: `task test`,
`task build-ex-qrcode`, and the full `task ci` gate passed against the local
ntpicture checkout with HEAD `8bf7a07`, using the explicit downstream-consumer
override. This includes race tests, raw Kitty PNG pixel equality/decoding,
custom glyph decoding, session-driven geometry, cancelled-job draining,
allocator failure/recovery, and cleanup after accepted transmission without
further owner routing. Native/Windows/WASM builds and the browser interaction
smoke test passed; both vulnerability scans reported no findings. No new
visual terminal/browser or camera acceptance is claimed for this lifecycle
change.


Atomic submission follow-up, 2026-10-14: adopted `HostGeometry: true` and
`Submit(Frame{Exact: true, Immutable: true, ...})` from sibling HEAD `1a86099`.
Removed the terminal-message filter and separate successful-layout clear/geometry
setters. Failed fits, empty content, and glyph transitions still clear explicitly.
`task test`, `task build-ex-qrcode`, and full `task ci` passed with the local
consumer override, including pixel/decoding regressions and browser interaction
smoke tests. Both vulnerability scans reported no findings. No additional manual
visual or camera checks were performed.


## Interactive generator demo

`task run-make` starts the text-to-QR TUI; `task build-ex-make` builds
`bin/ntqrcode-make`. The browser site's `demos/make.wasm` runs this same
application. The original `examples/qrcode` remains a two-component lifecycle
example and is served as the site's second demo, `demos/qrcode.wasm`.

The generator uses Bubbles textinput for Unicode editing, cursor navigation,
and paste. Enter generates the exact field value; Ctrl+U clears, Ctrl+G toggles
graphics, and Ctrl+C closes the component before quitting. Plain letters are
always input, including the original demo's q/g/n shortcuts. Any edit retires
the previous code, so an old payload never appears alongside new input. Empty,
over-limit, or unencodable payloads show an error without a stale code. Field
storage is bounded to 7,090 runes; encoding enforces the 7,089-byte limit and
QR capacity. The extra rune ensures a truncated oversized paste cannot become
a valid, silently shortened payload. This is a single-line text field.

Node 24+ is required for `task ci`. `task check-site` verifies `web/site.js`
syntax, the required booba runtime exports, every asset and local link that `web/index.html` references, and the
WASM headers of both demos, then runs `task smoke-wasm`, which exercises the
built `web/demos/make.wasm` through booba's real input/output bridge, checking
startup, paste, validation, generation, resize, and clean exit. It does not
inspect pixels.

Tests cover Unicode payload generation, ordinary typing, paste, invalid-input
recovery, bounded input, layout in half-block and solid-cell modes, and Kitty
retirement on editing and quitting. Run `task ci` against the pinned public
ntpicture dependency. Manual phone scanning and terminal/font
checks for the new generator remain outstanding.

Browser check, 2026-10-14: inspected the generator in Safari on macOS via the
local WASM site. Typing a replacement URL and pressing Enter displayed a new
Kitty QR. Switching to glyph mode removed the image and reported insufficient
space for that payload/geometry. No phone scan was performed.

Browser startup follow-up, 2026-10-14: upgraded go-booba assets from v0.7.0
to v0.9.2 to provide the action-bar API imported by the site. The site check
now links the required runtime exports. Full CI passed with the local ntpicture
consumer override. Verified in Safari that the generator loads and displays
the default QR code; no phone scan was performed.

Both demos use `examples/internal/demorun` to wrap their models with booba
`actions/bubblehelp`, following ntemoji. Bubbles key bindings advertise primary
controls and an expanded Quit action after capability negotiation. The page
renders them through its existing action bar; clicks send ordinary key events
to the same handlers as keyboard input. The WASM smoke test negotiates program
status and checks the generator advertises its controls.

Action-control verification, 2026-10-14: aligned the WASM Bubble Tea fork with
ntemoji (`c397c32a5074`) so OSC capability replies reach the wrapper. Full CI
passed, including the action negotiation smoke check. In Safari, verified both
demos display their buttons; Clear removed the generator text and QR, Generate
created the code for newly entered text, and Second code removed the second
pane. No phone scan was performed.

Example-module isolation verification, 2026-10-17: moved demos, shared action
runner, and the booba asset tool into `examples/go.mod`, following ntcharts-svg.
The root module no longer lists go-booba, Bubbles, Lip Gloss, clipboard, or pflag.
`task test`, `task build-ex-qrcode`, and full `task ci` passed across both modules,
including the downstream module-graph check, native/Windows/WASM builds, browser
smoke checks, and all three vulnerability scans (no findings). No additional
manual visual or camera checks were performed.
