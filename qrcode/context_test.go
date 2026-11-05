package qrcode

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEncodeContext(t *testing.T) {
	for _, payload := range []string{"hello", "日本語¢🙂", strings.Repeat("1", MaxPayloadBytes)} {
		opts := Options{Level: Low}
		want, err := Encode(payload, opts)
		if err != nil {
			t.Fatal(err)
		}
		got, err := EncodeContext(context.Background(), payload, opts)
		if err != nil || !reflect.DeepEqual(got.Matrix(), want.Matrix()) {
			t.Fatalf("context changed encoding: %v", err)
		}
	}
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := context.Canceled
		if deadline {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			want = context.DeadlineExceeded
		}
		cancel()
		code, err := EncodeContext(ctx, "hello", Options{})
		if code != nil || !errors.Is(err, want) || errors.Is(err, ErrCapacity) {
			t.Fatalf("cancellation: code=%v err=%v, want %v", code, err, want)
		}
	}
	if code, err := EncodeContext(context.Background(), strings.Repeat("x", MaxPayloadBytes+1), Options{}); code != nil || !errors.Is(err, ErrCapacity) {
		t.Fatalf("capacity: code=%v err=%v", code, err)
	}
}

// Cancel at a checkpoint rather than relying on goroutine scheduling or sleeps.
// The embedded context supplies the real cancellation error and closed Done.
type checkpointContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *checkpointContext) Err() error {
	c.checks--
	if c.checks == 0 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestEncodeCancellationDuringWork(t *testing.T) {
	for _, payload := range []string{strings.Repeat("1", MaxPayloadBytes), strings.Repeat("a¢日本語", 400)} {
		// Exercise cancellation after entry, including within segmentation.
		for _, checks := range []int{2, 10, 40} {
			base, cancel := context.WithCancel(context.Background())
			ctx := &checkpointContext{Context: base, cancel: cancel, checks: checks}
			code, err := EncodeContext(ctx, payload, Options{Level: Low})
			cancel()
			if code != nil || !errors.Is(err, context.Canceled) || errors.Is(err, ErrCapacity) {
				t.Fatalf("after %d checks: code=%v err=%v", checks, code, err)
			}
		}
	}
}
