// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ModelConfig is a getModelConfig result (firmware 1.22 and later; older
// firmware answers CodeMethodNotFound). Fields the answer lacks stay nil.
// Numbers are accepted for the flags, as devices send them so.
type ModelConfig struct {
	PS       *int  // ps
	PwmFreq  *int  // pwmFreq
	PwmRange []int // pwmRange
	WCR      *int  // wcr: white-to-colour ratio
	NoWC     *int  // nowc: number of white channels
	// CCTRange lists colour temperatures in kelvin, such as [2200, 2700,
	// 6500, 6500]; the range is its lowest to its highest (KelvinRange).
	CCTRange []int
	ExtRange []int // extRange, where present: the extended kelvin range
	FanSpeed *int  // fanSpeed: the number of fan speeds of a fan module
	PM       *int  // pm
	// HasAdjMinDim and HasTapSensor are hasAdjMinDim and hasTapSensor.
	HasAdjMinDim *bool
	HasTapSensor *bool
	ConfTs       *int // confTs
	MinDimLevel  *int // minDimLevel
	// Raw is every key as sent, including those not typed here:
	// renderFactor (a list in answers, a string in the vendor library's
	// setModelConfig), i2cTapSensor, drvIface, wizc1 and wizc2.
	Raw map[string]any
}

// ParseModelConfig reads a getModelConfig result.
func ParseModelConfig(b []byte) (ModelConfig, error) {
	var m ModelConfig
	if err := json.Unmarshal(b, &m.Raw); err != nil {
		return m, err
	}
	r := m.Raw
	m.PS, m.PwmFreq, m.PwmRange = intOf(r["ps"]), intOf(r["pwmFreq"]), intsOf(r["pwmRange"])
	m.WCR, m.NoWC = intOf(r["wcr"]), intOf(r["nowc"])
	m.CCTRange, m.ExtRange = intsOf(r["cctRange"]), intsOf(r["extRange"])
	m.FanSpeed, m.PM = intOf(r["fanSpeed"]), intOf(r["pm"])
	m.HasAdjMinDim, m.HasTapSensor = boolOf(r["hasAdjMinDim"]), boolOf(r["hasTapSensor"])
	m.ConfTs, m.MinDimLevel = intOf(r["confTs"]), intOf(r["minDimLevel"])
	return m, nil
}

// UserConfig is a getUserConfig result. Fields the answer lacks stay nil.
type UserConfig struct {
	FadeIn    *int  // fadeIn
	FadeOut   *int  // fadeOut
	FadeNight *bool // fadeNight
	DftDim    *int  // dftDim: default dimming
	PwmRange  []int // pwmRange
	// WhiteRange and ExtRange are kelvin ranges [min, max]; CCTRange is
	// read where a device reports it here.
	WhiteRange []int
	ExtRange   []int
	CCTRange   []int
	OpMode     *int // opMode
	// PO is po. Its meaning is not publicly described.
	PO       *bool
	FanSpeed *int // fanSpeed: the number of fan speeds, on older firmware
	// Raw is every key as sent.
	Raw map[string]any
}

// ParseUserConfig reads a getUserConfig result.
func ParseUserConfig(b []byte) (UserConfig, error) {
	var u UserConfig
	if err := json.Unmarshal(b, &u.Raw); err != nil {
		return u, err
	}
	r := u.Raw
	u.FadeIn, u.FadeOut, u.FadeNight, u.DftDim = intOf(r["fadeIn"]), intOf(r["fadeOut"]), boolOf(r["fadeNight"]), intOf(r["dftDim"])
	u.PwmRange, u.WhiteRange, u.ExtRange, u.CCTRange = intsOf(r["pwmRange"]), intsOf(r["whiteRange"]), intsOf(r["extRange"]), intsOf(r["cctRange"])
	u.OpMode, u.PO, u.FanSpeed = intOf(r["opMode"]), boolOf(r["po"]), intOf(r["fanSpeed"])
	return u, nil
}

// Default kelvin range for a light that reports none: openHAB's default.
const (
	DefaultKelvinMin = 2200
	DefaultKelvinMax = 6500
)

// KelvinRange is the colour-temperature range of a light: the lowest and
// highest value of the first list present among the model config's
// extRange and cctRange, then the user config's extRange and cctRange
// (the order pywizlight reads them in; pass a zero ModelConfig when the
// device has no getModelConfig). ok is false, with DefaultKelvinMin and
// DefaultKelvinMax, when none is present.
func KelvinRange(model ModelConfig, user UserConfig) (min, max int, ok bool) {
	for _, l := range [][]int{model.ExtRange, model.CCTRange, user.ExtRange, user.CCTRange} {
		if len(l) == 0 {
			continue
		}
		min, max = l[0], l[0]
		for _, v := range l[1:] {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		return min, max, true
	}
	return DefaultKelvinMin, DefaultKelvinMax, false
}

// FanSpeedMax is the number of fan speeds of a fan module: fanSpeed of the
// model config (firmware 1.22 and later), else of the user config, else
// DefaultFanSpeedMax. A fanSpeed of 0, which lights without a fan report,
// counts as absent.
func FanSpeedMax(model ModelConfig, user UserConfig) int {
	for _, n := range []*int{model.FanSpeed, user.FanSpeed} {
		if n != nil && *n > 0 {
			return *n
		}
	}
	return DefaultFanSpeedMax
}

// GetModelConfig asks a device for its model configuration.
func GetModelConfig(ctx context.Context, addr string, timeout time.Duration) (ModelConfig, error) {
	raw, err := Request(ctx, addr, MethodGetModelConfig, map[string]any{}, timeout)
	if err != nil {
		return ModelConfig{}, err
	}
	return ParseModelConfig(raw)
}

// GetUserConfig asks a device for its user configuration.
func GetUserConfig(ctx context.Context, addr string, timeout time.Duration) (UserConfig, error) {
	raw, err := Request(ctx, addr, MethodGetUserConfig, map[string]any{}, timeout)
	if err != nil {
		return UserConfig{}, err
	}
	return ParseUserConfig(raw)
}

// GetWifiConfig asks a device for its Wi-Fi configuration and returns the
// result as sent. Its content is not publicly described (openHAB notes it
// as an encrypted string, unverified).
func GetWifiConfig(ctx context.Context, addr string, timeout time.Duration) (json.RawMessage, error) {
	return Request(ctx, addr, MethodGetWifiConfig, map[string]any{}, timeout)
}

// UserConfigParams are the setUserConfig params the vendor's own library
// describes. Nil fields are left out; UserConfigTs is always sent (0 by
// default, as that library does).
type UserConfigParams struct {
	WhiteRange *[2]int // whiteRange: kelvin [min, max]
	ExtRange   *[2]int // extRange: extended kelvin [min, max]
	PwmRange   *[2]int // pwmRange: [min, max]
	DftDim     *int    // dftDim: default dimming, 1 to 100
	// StartupMode is sent as given when not empty; its values are not
	// publicly listed.
	StartupMode  string
	UserConfigTs int // userConfigTs
}

// Params returns the params. Ranges must not be negative or reversed.
func (u UserConfigParams) Params() (map[string]any, error) {
	p := map[string]any{}
	for key, r := range map[string]*[2]int{"whiteRange": u.WhiteRange, "extRange": u.ExtRange, "pwmRange": u.PwmRange} {
		if r == nil {
			continue
		}
		if r[0] < 0 || r[1] < r[0] {
			return nil, fmt.Errorf("%s %v is not a range [min, max]", key, *r)
		}
		p[key] = []int{r[0], r[1]}
	}
	if u.DftDim != nil {
		if *u.DftDim < 1 || *u.DftDim > 100 {
			return nil, fmt.Errorf("dftDim %d is outside 1 to 100", *u.DftDim)
		}
		p["dftDim"] = *u.DftDim
	}
	if u.StartupMode != "" {
		p["startupMode"] = u.StartupMode
	}
	if len(p) == 0 {
		return nil, errors.New("no user configuration given")
	}
	p["userConfigTs"] = u.UserConfigTs
	return p, nil
}

// SetUserConfig changes a device's user configuration.
func SetUserConfig(ctx context.Context, addr string, u UserConfigParams, timeout time.Duration) error {
	p, err := u.Params()
	if err != nil {
		return err
	}
	_, err = Request(ctx, addr, MethodSetUserConfig, p, timeout)
	return err
}

// SetSystemConfig sends setSystemConfig with params as given and returns
// the result. Dangerous: it changes factory and calibration settings
// (such as env, moduleName, ewf, pwmConf, drvConf, fs and systemConfigTs,
// the params the vendor's own library sends), and a wrong value can leave
// the driver misconfigured or the device unusable. The package offers no
// helpers for it because the safe values are not publicly described.
func SetSystemConfig(ctx context.Context, addr string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	if len(params) == 0 {
		return nil, errors.New("setSystemConfig: no params given")
	}
	return Request(ctx, addr, MethodSetSystemConfig, params, timeout)
}

// SetModelConfig sends setModelConfig with params as given (the
// getModelConfig keys; the vendor's own library requires confTs, pwmFreq,
// pwmRange, wcr, nowc, cctRange and renderFactor) and returns the result.
// Dangerous for the same reason as SetSystemConfig.
func SetModelConfig(ctx context.Context, addr string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	if len(params) == 0 {
		return nil, errors.New("setModelConfig: no params given")
	}
	return Request(ctx, addr, MethodSetModelConfig, params, timeout)
}

// intOf reads a JSON number as an int; nil for anything else.
func intOf(v any) *int {
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	n := int(f)
	return &n
}

// boolOf reads a JSON boolean, or a number (non-zero is true); nil for
// anything else.
func boolOf(v any) *bool {
	var b bool
	switch x := v.(type) {
	case bool:
		b = x
	case float64:
		b = x != 0
	default:
		return nil
	}
	return &b
}

// intsOf reads a JSON list of numbers; nil when v is not one.
func intsOf(v any) []int {
	l, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(l))
	for _, x := range l {
		n := intOf(x)
		if n == nil {
			return nil
		}
		out = append(out, *n)
	}
	return out
}
