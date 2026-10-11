// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Per-head control of dual-head lamps (DH in the module name).
//
// Some dual-head lamps split one light between their heads with ratio
// (RatioParams); others keep a state per head. For the second kind this
// package follows the numbering of pywizlight 0.6.6, whose author checked
// it on two dual-head lamps with firmware 1.38.0; it has not been checked
// against a real lamp for this package:
//
//   - a head is numbered 1 or 2;
//   - setPilot (and setState) carries the head in the devices key, 1 or 2;
//   - a getPilot for one head carries devices 0 or 1 (the head less one),
//     and the answer carries the head, 1 or 2, in its devices key;
//   - a syncPilot push about one head carries the head, 1 or 2, in its
//     devices key (Pilot.Head).
//
// Firmware that does not keep a state per head ignores the getPilot
// selector and answers with an untagged pilot; GetHeadPilot then returns
// ErrNoHeadState. Lamps that answer with ratio in their pilot split one
// light instead (Pilot.Ratio not nil). pywizlight's builder also accepts 3
// for the devices key, without describing it; this package does not send
// it.

// The heads of a dual-head lamp.
const (
	HeadFirst  = 1
	HeadSecond = 2
)

// ErrNoHeadState is returned by GetHeadPilot when the device answered with
// a pilot that is not tagged with the head asked for: it does not keep a
// state per head.
var ErrNoHeadState = errors.New("the device did not answer with that head's state")

func checkHead(head int) error {
	if head != HeadFirst && head != HeadSecond {
		return fmt.Errorf("head %d is neither 1 nor 2", head)
	}
	return nil
}

// HeadParams returns a copy of setPilot (or setState) params addressed to
// one head (HeadFirst or HeadSecond) of a dual-head lamp that keeps a
// state per head.
func HeadParams(head int, params map[string]any) (map[string]any, error) {
	if err := checkHead(head); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(params)+1)
	for k, v := range params {
		out[k] = v
	}
	out["devices"] = head
	return out, nil
}

// GetHeadPilot asks a dual-head lamp for one head's state (HeadFirst or
// HeadSecond): getPilot with devices set to the head less one. The answer
// must carry the head in its devices key; an untagged answer, or one about
// the other head, is ErrNoHeadState.
func GetHeadPilot(ctx context.Context, addr string, head int, timeout time.Duration) (Pilot, error) {
	if err := checkHead(head); err != nil {
		return Pilot{}, err
	}
	raw, err := Request(ctx, addr, MethodGetPilot, map[string]any{"devices": head - 1}, timeout)
	if err != nil {
		return Pilot{}, err
	}
	p, err := ParsePilot(raw)
	if err != nil {
		return Pilot{}, err
	}
	if p.Head() != head {
		return p, fmt.Errorf("getPilot for head %d: %w", head, ErrNoHeadState)
	}
	return p, nil
}

// Head returns the head (1 or 2) a pilot is about, from its devices key; 0
// when the pilot carries none or another value.
func (p Pilot) Head() int {
	if p.Devices == nil {
		return 0
	}
	switch *p.Devices {
	case HeadFirst:
		return HeadFirst
	case HeadSecond:
		return HeadSecond
	}
	return 0
}
