# Developing ntqrcode

Run `task ci` before submitting changes. The module follows the sibling
NTCharts projects: public component in `qrcode/`, a runnable example in
`examples/qrcode/`, native Task builds, and a booba WASM demo.

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
The host's picture component performs normal capability detection; the QR
component receives that choice and measured cell size. Unknown/unsupported
Kitty falls back to explicit black/white Unicode half-blocks built directly
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
preparation message instead of picture's resampled transitional glyph view.
Content/layout changes get fresh IDs and delete the retired image using
picture's existing commands and tmux handling. Presentation allocations are
bounded to 2,048-pixel edges, 256 Kitty cells per axis, or 65,536 glyph cells.

Encoder review (October 2026):

- [piglig/go-qr](https://github.com/piglig/go-qr) v2.6.0 is the encoder:
  MIT-licensed pure Go with no external runtime dependencies, implementing QR
  Model 2 versions 1–40, four correction levels, optimized mixed-mode
  segmentation, mask selection, and Reed–Solomon correction. It is a young,
  single-author project; we accept that risk because the decode-based tests
  below validate every symbol independently, and a differential corpus against
  skip2/go-qrcode (decode round-trips plus symbol sizes, 176 cases) matched.
  We use only its segment/matrix API and own the quiet zone, bounded image,
  and terminal presentation layers.
- The encoder is pinned to the upstream v2.6.0 release, which includes the
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

GitHub Actions uses `actions/checkout@v7`, `actions/setup-go@v7`, and
`go-task/setup-task@v2`, following the main NTCharts repo. CI invokes `task ci`
so local and hosted checks stay in sync. Hosted checks use Go 1.27.2. Both
`go.mod` and `wasm.work` prefer Go 1.27.2 or newer, so plain `task ci` selects
a patched toolchain with the default `GOTOOLCHAIN=auto` setting. Explicit
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
Pages builds on `main` pushes or manual dispatch, with deployment permissions
limited to the deploy job. Its build job runs the full `task ci` checks on
the selected revision before uploading the site, so automatic and manual
deployments require passing checks. Repository Pages settings must use
GitHub Actions.

## Releasing

The v0.2.0 module path is `nimbleterminal.dev/ntqrcode`; source and tags live
at `https://github.com/NimbleTerminal/ntqrcode`. Before publishing, ensure
`https://nimbleterminal.dev/ntqrcode?go-get=1` (and package subpaths) serves
HTML containing this metadata instead of redirecting to a marketing site:

```html
<meta name="go-import" content="nimbleterminal.dev/ntqrcode git https://github.com/NimbleTerminal/ntqrcode">
```

Update the dated changelog and dependency notices, run
`GOTOOLCHAIN=go1.27.2 task ci`, and commit the release changes. Publish the
version tag and GitHub release only after module discovery works. Then verify
`go get nimbleterminal.dev/ntqrcode/qrcode@v0.2.0` from a fresh module without
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

## Migrating from NTCharts/picture to ntpicture

This migration uses ntpicture's compatible low-level `Model`. The QR component
already owns rasterization, fit validation, replacement, and cleanup, so it does
not need to adopt the higher-level `Picture` lifecycle to change dependencies.
The exported `qrcode` API stays unchanged.

1. Inventory production, demo, and test imports. Replace
   `"github.com/NimbleMarkets/ntcharts/v2/picture"` with
   `picture "nimbleterminal.dev/ntpicture"`. The alias keeps existing call sites
   readable; the new package's declared name is `ntpicture`.
2. Rename `picture.NewWithConfig` to `picture.NewModelWithConfig`.
   For consumers using the default constructor, `picture.New` becomes
   `picture.NewModel`. The new `ntpicture.New` constructs a different,
   lifecycle-owning `Picture`; it is not the compatibility constructor.
3. Retain `Model`, `Config`, `KittyFrameMsg`, capability helpers, and the
   host-owned allocator. Keep asynchronous detection in the host and route
   frame messages to each QR component. Run all returned commands, including
   `Toggle`, replacement cleanup, and `Close`. Do not add synchronous probes.
4. Preserve the QR rendering contract: opaque black/white modules, four-module
   quiet zone, exact target-sized white padding, `FitFill`, transparent render
   background, and resolution factor 1. Keep direct module-to-glyph output and
   the preparation message; never show the renderer's transitional glyph view.
   Retain the accepted-frame cleanup guard and immediate retirement on invalid
   fit, replacement, resize, and removal.
5. Replace the module dependency, tidy, and regenerate dependency notices.
   Use a published tag or pseudo-version for a distributable library. A local
   `replace` is only a development bridge: Go ignores a dependency's replacement
   directives in downstream main modules. Do not weaken `TestDownstreamConsumer`
   to conceal an unpublished dependency. Native and WASM must resolve the same
   ntpicture revision; keep the existing WASM-only Bubble Tea/clipboard forks.
6. Run `task test`, `task build-ex-qrcode`, then `task ci`. Preserve the tests
   that independently decode actual Kitty PNG bytes and reconstructed glyphs,
   plus resize, capability, multi-component, and cleanup regressions. Regenerate
   notices with `task notices` before the CI notices check. Review the diff for
   stale imports, local replacements, and accidental dependency upgrades.

Adopting `Picture` or explicit `TerminalSession` is a separate lifecycle change.
`Picture` defaults to retaining the previous image while preparing a replacement;
QR needs `ClearOnReplace` and its own glyph policy. Its allocator also returns
`(int, error)`, while QR's existing `Config.NextID` returns `int`. Do not silently
change those contracts during an import migration.

Automated decoding and builds do not establish phone scanning or visual parity.
Native terminals, fonts, tmux, browser renderers, and camera scans still require
manual acceptance and a recorded result.

### Pre-release dependency and verification

The initial migration uses `nimbleterminal.dev/ntpicture v0.0.0` with
`replace nimbleterminal.dev/ntpicture => ../ntpicture`, as requested while the
sibling API is being finalized. This requires the sibling checkout for both
native and WASM builds and is not yet a distributable dependency pin. The
inspected sibling HEAD was `204e10268ffe4ee1785a606423a66df27483beb5`; the
replacement uses the working tree, including any subsequent local edits.

The downstream smoke test defaults to normal published dependency resolution.
For pre-release validation only, explicitly give its temporary main module the
same sibling path:

```sh
export NTQRCODE_TEST_NTPICTURE="$(cd ../ntpicture && pwd)"
task notices
task test
task --force build-ex-qrcode
task ci
```

Force the native example rebuild after sibling changes: Task tracks this
repository's sources, but not files in the replaced module.

This verifies a consumer with an explicit local dependency; it does not prove
that a remote consumer can fetch ntpicture. CI without the override will retain
that release-readiness check and cannot pass until the dependency is published.

Once ntpicture is released, remove the replacement and placeholder requirement,
then install the actual release (substitute its tag for `vX.Y.Z`):

```sh
go mod edit -dropreplace=nimbleterminal.dev/ntpicture
go get nimbleterminal.dev/ntpicture@vX.Y.Z
go mod tidy
unset NTQRCODE_TEST_NTPICTURE
task notices
task ci
```

Commit the new go.mod/go.sum and generated notices together. Run the consumer
test without the override before calling the migration release-ready.

Migration verification: `task test`, `task build-ex-qrcode`, and the full
`task ci` gate passed with the explicit local consumer override above. This
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
