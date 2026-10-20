# `ntqrcode` CHANGELOG

## Unreleased

* Upgrade dependencies: `piglig/go-qr` v2.6.0 to v2.7.0 (faster encoding,
  same segments and masks), `ntpicture` v0.1.0 to v0.1.1, and the demo
  module's `go-booba` v0.9.2 to v0.10.3. Regenerate dependency notices.

## v0.3.0 (2026-10-17)

* Retract `v0.1.0`–`v0.1.1` under the current module path because those tags
  declare the former `github.com/NimbleMarkets/ntcharts-qrcode` module path.
  Preserve the tags and their historical source.

* Isolate demo code and the booba asset tool in a separate `examples` module,
  removing go-booba and demo UI dependencies from the library module. Update
  Task/CI, dependency notices, and native/WASM checks to cover both modules.

* Advertise both demos’ keyboard controls through booba actions so browser users
  can generate, clear, toggle graphics, and control the two-code example with buttons.

* Upgrade booba assets to v0.9.2 so the demo action-bar imports resolve; validate
  runtime exports in the site check to catch startup failures.

* Replace the full-screen browser shell with the shared NimbleTerminal demo
  landing page: demo cards, titled terminal frame with Restart, booba action
  bar, status and help text, and footer license links. Serve both
  `examples/make` and `examples/qrcode` as `web/demos/<name>.wasm`, and add
  `task check-site` to validate site assets and links in CI.
* Add an interactive text/URL-to-QR generator for native terminals and the
  browser demo, with Unicode editing, paste, validation, and cleanup on edits.
  Keep the original two-component example available separately.

* Migrate rendering from `github.com/NimbleMarkets/ntcharts/v2/picture` to
  `nimbleterminal.dev/ntpicture.Picture`, delegating preparation, cancellation,
  presentation, and cleanup while preserving exact QR pixels and custom glyphs.
  Use `HostGeometry` and atomic `Submit` with exact, immutable rasters to prevent
  terminal events or intermediate geometry changes from resampling old pixels.
* Add optional `Config.Session` for host-owned terminal transport; both examples
  use explicit `TerminalSession` instances and close them after QR cleanup.
* **Behavior change:** `SetTerminal` now trusts the host's confirmed Kitty choice
  instead of requiring process-global detection. Hosts must combine their
  graphics preference with affirmative capability detection; rendering/allocation
  errors reach `Err`.
* **Breaking:** change `Config.NextID` to `func() (int, error)` to accept
  `ntpicture.NextImageID` directly. Preserve allocator errors through `Err()`
  and retain invalid/repeated-ID validation. Update both examples to use the
  shared allocator without adapters.
* Adopt the tagged ntpicture `v0.1.0` release, replacing the temporary sibling
  replacement and subsequent public commit pin. Native, WASM, and downstream
  consumer builds need no sibling checkout or test override.
* Record the migration procedure and verification in DEVELOP.md.
* Known limitation: a phone scan failed in a reported Apple Terminal solid-cell
  configuration. The supplied screenshot decodes automatically; camera acceptance
  for that configuration remains unresolved.

## v0.2.0 (2026-10-09)

* **Breaking:** move the module from `github.com/NimbleMarkets/ntcharts-qrcode`
  to `nimbleterminal.dev/ntqrcode`, hosted at
  [NimbleTerminal/ntqrcode](https://github.com/NimbleTerminal/ntqrcode).
  Replace imports with `nimbleterminal.dev/ntqrcode/qrcode`; the public API
  is unchanged.
* Rename the example binary to `bin/ntqrcode` and the browser demo prefix
  to `/ntqrcode/`.
* Update Go dependencies, including Bubble Tea `v2.1.0`, NTCharts `v2.7.2`,
  and `piglig/go-qr` `v2.6.0`; refresh demo dependency notices.
* Prefer Go `1.27.2` for native and WASM development and use it for hosted
  CI and Pages checks, including vulnerability scans. Plain `task ci` selects
  the patched toolchain with `GOTOOLCHAIN=auto`. The minimum supported Go
  version remains `1.26.8`.

## v0.1.1 (2026-10-07)

* Upgrade to [`piglig/go-qr`](https://github.com/piglig/go-qr) `v2.3.0`, including the version-search fix.
  Adapt segmentation to its new API while preserving ambiguous-Kanji byte
  encoding, UTF-8 ECI, version bounds, and correction boosting.

## v0.1.0 (2026-10-06)

Initial release. `ntqrcode` encodes QR codes and presents them inside
an existing Bubble Tea v2 event loop — with Kitty graphics or explicit
black/white Unicode half-blocks — as an experimental NTCharts companion
extracted from [gloss](https://github.com/NimbleMarkets/gloss).

* Bounded UTF-8 encoding, four correction levels, and configurable QR versions.
* Immutable module matrices and PNG-ready images with a four-module quiet zone.
* Composable Kitty and Unicode half-block rendering with fit checks,
  host-owned image IDs, and asynchronous cleanup.
* `Config.SolidCells` renders whole-cell modules for terminals, such as Apple
  Terminal, that misplace font-drawn half-block glyphs.
* Independent decoding tests, a two-code terminal example, and a WASM demo.
* Task-based CI and vulnerability scans, with Pages deployment gated on checks.
