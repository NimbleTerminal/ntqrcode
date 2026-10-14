# `ntqrcode` CHANGELOG

## Unreleased

* Migrate rendering from `github.com/NimbleMarkets/ntcharts/v2/picture` to
  `nimbleterminal.dev/ntpicture.Picture`, delegating preparation, cancellation,
  presentation, and cleanup while preserving exact QR pixels and custom glyphs.
  Use `HostGeometry` and atomic `Submit` with exact, immutable rasters to prevent
  terminal events or intermediate geometry changes from resampling old pixels.
* Add optional `Config.Session` for host-owned terminal transport; the terminal example
  uses an explicit `TerminalSession` instance and closes it after QR cleanup.
* **Behavior change:** `SetTerminal` now trusts the host's confirmed Kitty choice
  instead of requiring process-global detection. Hosts must combine their
  graphics preference with affirmative capability detection. The `NextID`
  callback signature remains unchanged; rendering/allocation errors reach `Err`.
* Use a temporary `../ntpicture` module replacement until its public release.
* Record the migration procedure and verification in DEVELOP.md.

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
