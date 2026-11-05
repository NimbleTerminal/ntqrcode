# Security posture

ntqrcode treats text, encoding options, and terminal dimensions as potentially
hostile inputs. Its defenses are input validation, bounded work and allocation,
cooperative cancellation, and keeping payload bytes out of terminal commands.
Encoding runs in the host process. This is not an OS security boundary or a
guarantee that the implementation and its dependencies contain no bugs.

## Reporting vulnerabilities

Please report vulnerabilities privately to **security@neomantra.net**, following
the sibling NimbleTerminal projects' reporting convention. Include the affected
revision, Go version, platform, and a minimal reproduction. Do not include
credentials or private QR payloads. Avoid public issues containing exploit details
before maintainers have had an opportunity to investigate. Security fixes target
the current release; this experimental project has no maintained older branches.

## Production perimeter

The `qrcode` package converts a UTF-8 string into an immutable module matrix and
then into generated pixels or terminal glyphs. It does not fetch URLs, execute
payloads, read files, accept compressed images, or decode QR symbols. The
independent QR decoder and PNG decoding used to verify output are test-only.
The upstream go-qr dependency also offers decoding APIs; ntqrcode does not call
or expose them.

ntpicture receives a newly constructed, bounded raster through its exact-size
submission path. We do not import its image decoders or `decoders/worker`.
The presence of ntmediabox in ntpicture's module graph does not mean ntqrcode
runs encoding or rendering in a sandbox. There is no worker or IPC protocol here.

If production image or QR decoding is added, review that new boundary explicitly:
cap compressed input before buffering, validate dimensions and pixel counts
before allocation, and use a confined worker with validated output and process
termination for hard timeouts. A context alone does not isolate a decoder.

## Bounds and allocation

Checks precede the corresponding expensive work or allocation:

| Input or output | Enforced limit |
| --- | --- |
| Text | Nonempty, valid UTF-8, at most 7,089 bytes before segmentation |
| Correction and version options | Four supported correction levels; versions 1–40 with ordered bounds; zero version values select defaults |
| Module matrix | At most 185 × 185 including the four-module quiet zone |
| Exported image | Integer scale 1–16 and at most 2,048 pixels on either side |
| Terminal cell geometry | Each dimension 1–256 pixels |
| Kitty placement | At most 256 cells on either axis and at most 2,048 pixels on either axis |
| Glyph output | At most 65,536 cells, checked before constructing the string |
| Image identifiers | Positive 24-bit IDs; allocation errors and immediately repeated IDs are rejected |

Segmentation uses a fixed four-mode dynamic program and at most three QR
version ranges. The byte limit bounds its rune and segment buffers. Image
scale and terminal cell dimensions are checked before multiplication by
untrusted values; Kitty dimensions are clamped before cell-to-pixel conversion.
The maximum exported grayscale image has 4 MiB of pixel storage; a maximum
RGBA presentation raster has 16 MiB. These figures are per raster, not a total
process memory budget: matrices, strings, PNG buffers, dependency scratch space,
and pending presentation work add overhead.

No part of the quiet zone is cropped and pixels are not resampled. Actual Kitty
PNG decoding tests protect the dependency's exact-size rendering contract.

## Cancellation and host responsibilities

`EncodeContext(ctx, text, opts)` is synchronous and cooperatively cancelable.
It checks the context at entry, throughout segmentation, between encoding
stages, and while copying the final matrix. Cancellation returns a nil code and
`context.Canceled` or `context.DeadlineExceeded`, recognizable with `errors.Is`;
these are not reported as `ErrCapacity`. `Encode` delegates to it with a
background context and has no cancellation deadline.

Upstream `EncodeSegments` has no context API. An in-flight call completes before
our next cancellation check. Its input and QR version are bounded, but there is
no promised maximum cancellation latency. Cancellation does not start or abandon
a background goroutine. A timer around a goroutine would only stop waiting and
would leave encoding work running; that is not an execution limit. Applications
requiring enforceable deadlines or containment must use a separately managed
process with OS resource limits and termination.

Hosts must bound request queues, simultaneous encodes, retained codes and images,
and input buffering before calling this API. A per-call input limit does not
prevent aggregate exhaustion. Keep expensive work off the UI event loop when
responsiveness matters; cancel superseded requests, wait for their work to stop,
and discard stale results. Calls with independent inputs may run concurrently;
a terminal `Model` belongs to one event loop and must not be copied or mutated
concurrently. The terminal host must execute every returned command, drain
messages, and run `Close()` cleanup before discarding a model.

Terminal output is constructed from fixed glyphs, colors, numeric graphics
parameters, and generated raster bytes. Raw payload text is not interpolated
into that output. Applications that separately echo or log input must handle
control characters in those paths. Custom allocator errors and host callbacks
are trusted application inputs, not a hostile-peer protocol. The host also owns
terminal capability detection and globally fresh image IDs; checking the last
ID is not a registry of every ID ever allocated. Terminal emulators, browser
runtimes, and rendering dependencies remain part of the trusted computing base.

## Verification and limits of the evidence

`task ci` runs formatting and module checks, race tests, vet, native/Windows/WASM
builds, browser interaction smoke tests, dependency notice checks, vulnerability
scans for native and WASM graphs, and bounded fuzz campaigns. Fuzz seeds also run
as ordinary regression tests. `task fuzz FUZZTIME=1m` runs longer local campaigns.

Fuzz targets exercise arbitrary bytes and Unicode, invalid options, extreme
integers, image scale, and terminal geometry. They check matrix and allocation
bounds, quiet zones, independent decoding of generated images, and exclusion of
payload controls from glyph output. Deterministic tests cover canceled and
expired contexts and cancellation after encoding work has begun. Existing
regressions decode actual transmitted Kitty PNGs and reconstructed glyphs.

These checks can find implementation bugs and known dependency advisories;
they do not prove absence of vulnerabilities, hard CPU limits, terminal visual
correctness, or camera acceptance. Manual acceptance results are recorded
separately in DEVELOP.md. Preserve these limits and tests when changing the
encoder, segmentation, presentation, or dependencies.
