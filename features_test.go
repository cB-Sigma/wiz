// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Covers: SIG-IPE-4341
func TestClassOfFansAndOrder(t *testing.T) {
	// The second segment is checked for RGB, DDTW, TW, SOCKET and FANDIM in
	// that order; a segment holding two of them takes the first.
	for module, want := range map[string]Class{
		"ESP03_FANDIMS_31": ClassFanDim, "ESP03_FANDIMS_01": ClassFanDim, "ESP20_DDTW1_01": ClassFanTW,
		"ESP20_DHRGB_01": ClassRGB, "ESP99_DDTWRGB_01": ClassRGB, "ESP99_FANDIMTW_01": ClassTW,
		"ESP99_SOCKETFANDIM_01": ClassSocket, "esp03_fandims_31": ClassFanDim,
	} {
		if c, ok := ClassOf(module); !ok || c != want {
			t.Errorf("%s: %s %v, want %s", module, c, ok, want)
		}
	}
	if c, ok := ClassOfTypeID(0); !ok || c != ClassDW {
		t.Errorf("typeId 0 = %s %v", c, ok)
	}
	if c, ok := ClassOfTypeID(7); ok || c != ClassUnknown {
		t.Errorf("typeId 7 = %s %v", c, ok)
	}
	tid := 0
	if c, ok := ClassOfConfig(SystemConfig{TypeID: &tid}); !ok || c != ClassDW {
		t.Errorf("config without a module = %s %v", c, ok)
	}
	if c, ok := ClassOfConfig(SystemConfig{ModuleName: "ESP20_DDTW1_01", TypeID: &tid}); !ok || c != ClassFanTW {
		t.Errorf("config with a module = %s %v", c, ok)
	}
	if c, ok := ClassOfConfig(SystemConfig{}); ok || c != ClassUnknown {
		t.Errorf("empty config = %s %v", c, ok)
	}
	// Fan lights have no published scene list.
	if ScenesFor(ClassFanDim) != nil || HasScene(ClassFanTW, 11) {
		t.Error("scenes for a fan light")
	}
}

func TestFeaturesOf(t *testing.T) {
	for module, want := range map[string]Features{
		"ESP01_SHRGB1C_31":   {Brightness: true, Color: true, ColorTemp: true, Effects: true},
		"ESP20_DHRGB_01":     {Brightness: true, Color: true, ColorTemp: true, Effects: true, DualHead: true},
		"ESP56_SHTW3_01":     {Brightness: true, ColorTemp: true, Effects: true},
		"ESP05_SHDW9_01":     {Brightness: true, Effects: true},
		"ESP01_DIMTRIACS_01": {},
		"ESP06_DW1_01":       {Brightness: true},
		"ESP10_SOCKET_06":    {},
		"ESP03_FANDIMS_31":   {Brightness: true, Fan: true, FanBreeze: true, FanReverse: true},
		"ESP20_DDTW1_01":     {Brightness: true, ColorTemp: true, Effects: true, Fan: true, FanBreeze: true, FanReverse: true},
		"ESP99_XYZ_01":       {},
	} {
		if got := FeaturesOf(module); got != want {
			t.Errorf("%s: %+v, want %+v", module, got, want)
		}
	}
}

func TestFanParams(t *testing.T) {
	on, rev := true, false
	p, err := FanParams{On: &on, Mode: FanModeBreeze, Speed: 4, Reverse: &rev}.Params(6)
	if err != nil || !reflect.DeepEqual(p, map[string]any{"fanState": 1, "fanMode": 2, "fanSpeed": 4, "fanRevrs": 0}) {
		t.Errorf("params %v %v", p, err)
	}
	off := false
	if p, err := (FanParams{On: &off}).Params(0); err != nil || !reflect.DeepEqual(p, map[string]any{"fanState": 0}) {
		t.Errorf("off %v %v", p, err)
	}
	// Without a known maximum, speeds up to 6 are accepted.
	if _, err := (FanParams{Speed: 6}).Params(0); err != nil {
		t.Errorf("speed 6 with the default maximum: %v", err)
	}
	for _, bad := range []FanParams{{}, {Speed: 7}, {Speed: -1}, {Mode: 3}, {Mode: -1}} {
		if p, err := bad.Params(6); err == nil {
			t.Errorf("%+v accepted: %v", bad, p)
		}
	}
	if _, err := (FanParams{Speed: 9}).Params(10); err != nil {
		t.Errorf("speed 9 of 10: %v", err)
	}
	if p, err := RatioParams(40); err != nil || p["ratio"] != 40 {
		t.Errorf("ratio %v %v", p, err)
	}
	for _, bad := range []int{-1, 101} {
		if _, err := RatioParams(bad); err == nil {
			t.Errorf("ratio %d accepted", bad)
		}
	}
}

func TestParsePilotFanRatioPower(t *testing.T) {
	p, err := ParsePilot([]byte(`{"mac":"a8bb50e4c1f1","state":false,"dimming":40,"fanState":1,"fanMode":2,"fanSpeed":3,"fanRevrs":1,"ratio":30,"pc":5200}`))
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]*float64{"fanState": p.FanState, "fanMode": p.FanMode, "fanSpeed": p.FanSpeed, "fanRevrs": p.FanRevrs, "ratio": p.Ratio, "pc": p.PC} {
		if got == nil || *got != p.Raw[name].(float64) {
			t.Errorf("%s = %v", name, got)
		}
	}
	// The light and the fan are independent: the light is off, the fan on.
	if p.State == nil || *p.State || *p.FanState != 1 {
		t.Errorf("pilot %+v", p)
	}
	b, _ := json.Marshal(Pilot{FanSpeed: p.FanSpeed})
	if string(b) != `{"fanSpeed":3}` {
		t.Errorf("marshalled %s", b)
	}
}
