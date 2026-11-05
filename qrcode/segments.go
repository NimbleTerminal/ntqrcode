package qrcode

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	encoder "github.com/piglig/go-qr/v2"
)

// These JIS X 0208 characters map to different Unicode characters in common
// Shift-JIS decoders (including ZXing). Keep them out of Kanji segments.
const ambiguousKanji = "\\¢£¬‖−〜"

func encodeSymbol(ctx context.Context, content string, level encoder.ECC, minV, maxV int) (*encoder.Code, error) {
	var prefix []encoder.Segment
	for _, r := range content {
		if r >= utf8.RuneSelf {
			eci, err := encoder.ECISegment(26) // Explicit UTF-8 for every byte segment.
			if err != nil {
				return nil, err
			}
			prefix = append(prefix, eci)
			break
		}
	}
	// Each candidate appends to the same ECI prefix; prevent one append from
	// overwriting another candidate's segments through spare slice capacity.
	prefix = prefix[:len(prefix):len(prefix)]

	// Splitting around ambiguous characters can add more headers than a single
	// byte segment. Retain that alternative, so segmentation cannot reject text
	// that fits in byte mode or inflate its symbol size.
	var fallback *encoder.Code
	var byteSegments []encoder.Segment
	if strings.ContainsAny(content, ambiguousKanji) {
		segment := encoder.BytesSegment([]byte(content))
		var err error
		byteSegments = append(prefix, segment)
		fallback, err = encodeSegments(ctx, byteSegments, level, minV, maxV)
		if err != nil && !errors.Is(err, encoder.ErrDataTooLong) {
			return nil, err
		}
		if fallback != nil {
			maxV = (fallback.Size() - 17) / 4
		}
	}

	// Character-count widths, and hence optimal segmentation, only change at
	// versions 10 and 27. Segment at each range's end, then let EncodeSegments
	// perform the bounded capacity search including our ECI header, without
	// repeating segmentation at every version.
	for _, boundary := range [...]int{9, 26, 40} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(boundary, maxV)
		if end < minV {
			continue
		}
		segments, err := safeSegments(ctx, content, end)
		if err == nil {
			segments = append(prefix, segments...)
			var qr *encoder.Code
			qr, err = encodeSegments(ctx, segments, level, minV, end)
			if err == nil {
				if fallback != nil && fallback.Size() == qr.Size() {
					if fallback.ECC() > qr.ECC() {
						return fallback, nil
					}
				}
				return qr, nil
			}
		}
		if !errors.Is(err, encoder.ErrDataTooLong) {
			return nil, err
		}
		minV = end + 1
	}
	if fallback != nil {
		return fallback, nil
	}
	return nil, encoder.ErrDataTooLong
}

// Optimize safe runs and encode consecutive ambiguous characters as UTF-8.
// Splitting only at those boundaries preserves compression of long numeric,
// alphanumeric, and Japanese runs surrounding an ambiguous character.
func safeSegments(ctx context.Context, content string, version int) ([]encoder.Segment, error) {
	var segments []encoder.Segment
	for len(content) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := strings.IndexAny(content, ambiguousKanji)
		if end < 0 {
			end = len(content)
		}
		if end > 0 {
			run, err := optimalSegments(ctx, content[:end], version)
			if err != nil {
				return nil, err
			}
			segments = append(segments, run...)
		} else {
			for _, r := range content {
				if !strings.ContainsRune(ambiguousKanji, r) {
					break
				}
				end += utf8.RuneLen(r)
			}
			run := encoder.BytesSegment([]byte(content[:end]))
			segments = append(segments, run)
		}
		content = content[end:]
	}
	return segments, nil
}

// Upstream has no context-aware segment encoder. Check on both sides rather
// than returning early and leaving its CPU/memory work running in a goroutine.
func encodeSegments(ctx context.Context, segments []encoder.Segment, level encoder.ECC, minV, maxV int) (*encoder.Code, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	code, err := encoder.EncodeSegments(segments, encoder.WithECC(level), encoder.WithVersionRange(minV, maxV))
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	return code, err
}
