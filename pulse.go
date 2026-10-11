// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"context"
	"fmt"
	"time"
)

// MaxPulseDelta is the largest brightness change, in percent, a pulse
// takes in either direction.
const MaxPulseDelta = 100

// PulseParams returns the params of a pulse: delta is the brightness
// change in percent, -100 to 100 and not 0 (negative fades out, positive
// fades in), and duration how long the pulse lasts, sent in whole
// milliseconds (at least 1 ms). pywizlight's README describes these two
// params; it also says a pulse longer than two minutes repeats until the
// light is set again. That has not been checked against a real device.
func PulseParams(delta int, duration time.Duration) (map[string]any, error) {
	if delta == 0 || delta < -MaxPulseDelta || delta > MaxPulseDelta {
		return nil, fmt.Errorf("pulse delta %d is not -100 to 100 without 0", delta)
	}
	ms := duration.Milliseconds()
	if ms < 1 {
		return nil, fmt.Errorf("pulse duration %s is shorter than 1 ms", duration)
	}
	return map[string]any{"delta": delta, "duration": ms}, nil
}

// Pulse sends a pulse (see PulseParams) and waits for the answer until
// timeout. It is sent once, not resent: a pulse is not idempotent, and a
// resend that crossed a lost answer would pulse the light twice.
func Pulse(ctx context.Context, addr string, delta int, duration time.Duration, timeout time.Duration) error {
	p, err := PulseParams(delta, duration)
	if err != nil {
		return err
	}
	_, err = request(ctx, addr, MethodPulse, p, timeout, false)
	return err
}
