// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"errors"
	"fmt"
	"strings"
)

// Features are what a module offers, told from its module name the way
// public clients tell it (see PROVENANCE.md).
type Features struct {
	Brightness bool // dimming
	Color      bool // r, g, b (and c, w)
	ColorTemp  bool // temp
	// Effects: public clients offer scenes (sceneId, with speed) for it.
	// Dimmable-white modules have them only when the second segment holds
	// SH or DH; fan modules with a dimmable light have none.
	Effects    bool
	Fan        bool // fanState, fanSpeed
	FanBreeze  bool // fanMode 2
	FanReverse bool // fanRevrs
	// DualHead: two light heads (DH in the second segment), whose split is
	// set with ratio (RatioParams). Some dual-head lights do not say so in
	// their name; Home Assistant also treats a device whose pilot carries
	// ratio (Pilot.Ratio not nil) as one.
	DualHead bool
}

// FeaturesOf tells a module's features from its name. A module ClassOf
// does not know has none.
func FeaturesOf(module string) Features {
	c, ok := ClassOf(module)
	if !ok {
		return Features{}
	}
	seg, _ := segment(module)
	f := Features{DualHead: strings.Contains(seg, "DH")}
	switch c {
	case ClassRGB:
		f.Brightness, f.Color, f.ColorTemp, f.Effects = true, true, true, true
	case ClassTW:
		f.Brightness, f.ColorTemp, f.Effects = true, true, true
	case ClassDW:
		f.Brightness = true
		f.Effects = strings.Contains(seg, "SH") || strings.Contains(seg, "DH")
	case ClassFanDim:
		f.Brightness, f.Fan, f.FanBreeze, f.FanReverse = true, true, true, true
	case ClassFanTW:
		f.Brightness, f.ColorTemp, f.Effects, f.Fan, f.FanBreeze, f.FanReverse = true, true, true, true, true, true
	}
	return f
}

// FanMode is a fan module's mode (the fanMode key).
type FanMode int

// The fan modes.
const (
	FanModeNormal FanMode = 1
	FanModeBreeze FanMode = 2 // speed varies, like a breeze
)

// DefaultFanSpeedMax is the number of fan speeds assumed when the device
// does not report one (see FanSpeedMax).
const DefaultFanSpeedMax = 6

// FanParams sets a fan module's fan, as setPilot (or setState) params. Zero
// fields are left out, so the device keeps their value. The fan is
// independent of the module's light: switching one does not switch the
// other.
type FanParams struct {
	On      *bool   // fanState: 1 on, 0 off
	Mode    FanMode // fanMode; 0 leaves it
	Speed   int     // fanSpeed, 1 to the module's maximum; 0 leaves it
	Reverse *bool   // fanRevrs: 1 reverse, 0 forward
}

// Params returns the params, checking the speed against maxSpeed (from
// FanSpeedMax; DefaultFanSpeedMax when maxSpeed is 0 or less). Home
// Assistant sends FanModeNormal with a speed, and FanModeBreeze alone. To
// stop the fan send On false; openHAB's speed 0 means the same, but this
// package does not send speed 0.
func (f FanParams) Params(maxSpeed int) (map[string]any, error) {
	if maxSpeed <= 0 {
		maxSpeed = DefaultFanSpeedMax
	}
	p := map[string]any{}
	if f.On != nil {
		p["fanState"] = b01(*f.On)
	}
	switch f.Mode {
	case 0:
	case FanModeNormal, FanModeBreeze:
		p["fanMode"] = int(f.Mode)
	default:
		return nil, fmt.Errorf("fan mode %d is neither 1 (normal) nor 2 (breeze)", f.Mode)
	}
	if f.Speed != 0 {
		if f.Speed < 1 || f.Speed > maxSpeed {
			return nil, fmt.Errorf("fan speed %d is outside 1 to %d", f.Speed, maxSpeed)
		}
		p["fanSpeed"] = f.Speed
	}
	if f.Reverse != nil {
		p["fanRevrs"] = b01(*f.Reverse)
	}
	if len(p) == 0 {
		return nil, errors.New("no fan setting given")
	}
	return p, nil
}

func b01(b bool) int {
	if b {
		return 1
	}
	return 0
}

// RatioParams sets the split of a dual-head module's light between its up
// and down heads, 0 to 100, as setPilot params.
func RatioParams(ratio int) (map[string]any, error) {
	if ratio < 0 || ratio > 100 {
		return nil, fmt.Errorf("ratio %d is outside 0 to 100", ratio)
	}
	return map[string]any{"ratio": ratio}, nil
}
