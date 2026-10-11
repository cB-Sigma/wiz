// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"reflect"
	"testing"
)

func TestParseModelConfig(t *testing.T) {
	// Booleans may come as numbers, renderFactor as a list; keys this
	// package does not type stay in Raw.
	m, err := ParseModelConfig([]byte(`{"ps":1,"pwmFreq":1000,"pwmRange":[0,100],"wcr":30,"nowc":2,"cctRange":[2200,2700,6500,6500],
		"renderFactor":[255,0,255,255,0,0,0,0,0,0],"fanSpeed":5,"pm":0,"hasAdjMinDim":0,"hasTapSensor":true,"drvIface":0,"wizc1":{"mode":[0]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if *m.PS != 1 || *m.PwmFreq != 1000 || *m.WCR != 30 || *m.NoWC != 2 || *m.FanSpeed != 5 || *m.PM != 0 ||
		*m.HasAdjMinDim || !*m.HasTapSensor || !reflect.DeepEqual(m.CCTRange, []int{2200, 2700, 6500, 6500}) ||
		!reflect.DeepEqual(m.PwmRange, []int{0, 100}) || m.ExtRange != nil || m.ConfTs != nil || m.Raw["wizc1"] == nil || m.Raw["drvIface"] == nil {
		t.Errorf("model config %+v", m)
	}
	if _, err := ParseModelConfig([]byte(`[1]`)); err == nil {
		t.Error("a list parsed as a model config")
	}
}

func TestParseUserConfig(t *testing.T) {
	u, err := ParseUserConfig([]byte(`{"fadeIn":400,"fadeOut":600,"fadeNight":false,"dftDim":90,"pwmRange":[0,100],"whiteRange":[2700,6500],"extRange":[2200,6500],"opMode":0,"po":true,"fanSpeed":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if *u.FadeIn != 400 || *u.FadeOut != 600 || *u.FadeNight || *u.DftDim != 90 || *u.OpMode != 0 || !*u.PO || *u.FanSpeed != 4 ||
		!reflect.DeepEqual(u.WhiteRange, []int{2700, 6500}) || !reflect.DeepEqual(u.ExtRange, []int{2200, 6500}) || u.Raw["po"] != true {
		t.Errorf("user config %+v", u)
	}
}

func TestKelvinRangeAndFanSpeedMax(t *testing.T) {
	for name, c := range map[string]struct {
		model    ModelConfig
		user     UserConfig
		min, max int
		ok       bool
	}{
		"model extRange first": {ModelConfig{ExtRange: []int{2000, 6000}, CCTRange: []int{2200, 6500}}, UserConfig{ExtRange: []int{2700, 5000}}, 2000, 6000, true},
		"model cctRange":       {ModelConfig{CCTRange: []int{2200, 2700, 6500, 6500}}, UserConfig{ExtRange: []int{2700, 5000}}, 2200, 6500, true},
		"user extRange":        {ModelConfig{}, UserConfig{ExtRange: []int{2700, 5000}, CCTRange: []int{1, 2}}, 2700, 5000, true},
		"user cctRange":        {ModelConfig{}, UserConfig{CCTRange: []int{6500, 2700}}, 2700, 6500, true},
		"defaults":             {ModelConfig{}, UserConfig{WhiteRange: []int{2700, 6500}}, 2200, 6500, false},
	} {
		min, max, ok := KelvinRange(c.model, c.user)
		if min != c.min || max != c.max || ok != c.ok {
			t.Errorf("%s: %d..%d %v", name, min, max, ok)
		}
	}
	four, zero, three := 4, 0, 3
	for _, c := range []struct {
		model ModelConfig
		user  UserConfig
		want  int
	}{
		{ModelConfig{FanSpeed: &four}, UserConfig{FanSpeed: &three}, 4},
		{ModelConfig{FanSpeed: &zero}, UserConfig{FanSpeed: &three}, 3},
		{ModelConfig{}, UserConfig{}, DefaultFanSpeedMax},
	} {
		if got := FanSpeedMax(c.model, c.user); got != c.want {
			t.Errorf("FanSpeedMax(%+v, %+v) = %d, want %d", c.model, c.user, got, c.want)
		}
	}
}

func TestUserConfigParams(t *testing.T) {
	dim := 80
	p, err := UserConfigParams{WhiteRange: &[2]int{2700, 6500}, DftDim: &dim, StartupMode: "x"}.Params()
	want := map[string]any{"whiteRange": []int{2700, 6500}, "dftDim": 80, "startupMode": "x", "userConfigTs": 0}
	if err != nil || !reflect.DeepEqual(p, want) {
		t.Errorf("params %v %v", p, err)
	}
	p, _ = UserConfigParams{ExtRange: &[2]int{2200, 6500}, PwmRange: &[2]int{5, 100}, UserConfigTs: 17}.Params()
	if !reflect.DeepEqual(p, map[string]any{"extRange": []int{2200, 6500}, "pwmRange": []int{5, 100}, "userConfigTs": 17}) {
		t.Errorf("params %v", p)
	}
	zero, big := 0, 101
	for _, bad := range []UserConfigParams{{}, {WhiteRange: &[2]int{6500, 2700}}, {PwmRange: &[2]int{-1, 100}}, {DftDim: &zero}, {DftDim: &big}} {
		if p, err := bad.Params(); err == nil {
			t.Errorf("%+v accepted: %v", bad, p)
		}
	}
}

func TestParseSystemConfigExtras(t *testing.T) {
	c, err := ParseSystemConfig([]byte(`{"mac":"A8:BB:50:E4:C1:01","homeId":1,"homeLock":false,"pairingLock":true,"typeId":0,"moduleName":"ESP01_SHRGB1C_31",
		"fwVersion":"1.28.0","groupId":3,"drvConf":[20,2],"ewf":[255,0,255,255,0,0,0],"ewfHex":"ff00ffff000000","ping":0,"rgn":"eu","lock":0,"fwOtaStatus":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.MAC != "a8bb50e4c101" || *c.HomeLock || !*c.PairingLock || *c.Lock || *c.GroupID != 3 || *c.WhiteToColorRatio != 20 || *c.WhiteChannels != 2 ||
		c.EWFHex != "ff00ffff000000" || *c.Ping != 0 || c.Region != "eu" || *c.FwOtaStatus != 2 {
		t.Errorf("system config %+v", c)
	}
	// Older answers lack the extras; they stay unset.
	c, _ = ParseSystemConfig([]byte(`{"mac":"a8bb50e4c101","moduleName":"ESP05_SHDW9_01"}`))
	if c.HomeLock != nil || c.WhiteChannels != nil || c.FwOtaStatus != nil || c.Region != "" {
		t.Errorf("system config %+v", c)
	}
}
