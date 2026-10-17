# ntqrcode

An experimental NTCharts companion for QR codes in Bubble Tea v2, extracted
from gloss. Module: `nimbleterminal.dev/ntqrcode`, Go 1.26.8+.

## Layout

- `qrcode`: encoding, image generation, and terminal component. No application dependencies.
- `examples/go.mod`: separate demo module; owns go-booba and the asset tool.
- `examples/internal/demorun`: shared booba action wrapper for demos.
- `examples/qrcode`: two-component native example using the public API.
- `examples/make`: interactive QR generator shared by native and WASM demos.
- `web`: booba demo landing page (`index.html`, `site.css`, `site.js` tracked);
  `demos/`, runtime, and license copies are generated and ignored.
- `wasm.work`: library + examples workspace, with Bubble Tea/clipboard forks for WASM only.
- `internal/tools/notices`: generates demo dependency licenses with `task notices`.

## Conventions

Keep the public API small. Encoding and presentation are separate types.
The component never reads stdin, takes over the terminal, or synchronously
probes capabilities. Hosts own the event loop, terminal detection, and the
shared image-ID allocator. Run every returned command, including `Close()`.

Keep black modules on opaque white with a four-module quiet zone. Never crop
a QR or filter its edges. Kitty and image modules are exactly square; glyphs
may approximate the cell geometry within the documented 9/8 side ratio.
Preserve allocation and input bounds. Sixel, logos, styling, and product
decoding are out of scope. The decoder dependency is for tests only.

ntpicture's exact-size `FitFill` path must bypass resampling. Keep the
tests that decode actual Kitty PNG bytes and reconstructed glyph output;
they protect this dependency contract when upgrading ntpicture.

## Verification

Run `task ci` before finishing: formatting, tidy/verify, race tests, vet,
Windows build, native examples, WASM build, and native/WASM vulnerability
scans (network required). Use `task test` and
`task build-ex-qrcode` during development. Follow the sibling NTCharts
projects' MIT license, Task, examples, and documentation conventions.

Keep README, DEVELOP, and CHANGELOG consistent. Never represent automated
decoding as an actual phone scan. Terminal/font/tmux and camera checks remain
manual acceptance checks until someone performs and records them.
