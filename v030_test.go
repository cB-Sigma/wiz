// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cB-Sigma/wiz"
)

func near(a, b int) bool { return a-b <= 1 && b-a <= 1 }

func TestPywizlightColour(t *testing.T) {
	for _, c := range []struct {
		hue, sat float64
		want     wiz.RGBCW
	}{
		{0, 1, wiz.RGBCW{R: 255}},
		{360, 1, wiz.RGBCW{R: 255}},
		{120, 0.75, wiz.RGBCW{G: 255, W: 64}},
		{240, 0.25, wiz.RGBCW{B: 127, W: 128}},
		{-120, 0.5, wiz.RGBCW{B: 255, W: 128}},
		{77, 0, wiz.RGBCW{W: 128}},
		{60, 1, wiz.RGBCW{R: 255, G: 255}},
		{180, 1, wiz.RGBCW{G: 255, B: 255}},
		{30, 1, wiz.RGBCW{R: 255, G: 127}},
	} {
		got, err := wiz.HSToRGBWPywizlight(c.hue, c.sat)
		if err != nil {
			t.Fatal(err)
		}
		if !near(got.R, c.want.R) || !near(got.G, c.want.G) || !near(got.B, c.want.B) || got.W != c.want.W || got.C != 0 {
			t.Errorf("HSToRGBWPywizlight(%v, %v) = %+v, want %+v (each colour channel within 1)", c.hue, c.sat, got, c.want)
		}
	}
	for _, bad := range [][2]float64{{0, -0.1}, {0, 1.1}} {
		if _, err := wiz.HSToRGBWPywizlight(bad[0], bad[1]); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	for _, c := range []struct {
		r, g, b int
		want    wiz.RGBCW
	}{
		{255, 0, 0, wiz.RGBCW{R: 255}},
		{0, 0, 255, wiz.RGBCW{B: 255}},
		{255, 255, 255, wiz.RGBCW{W: 128}},
		{0, 0, 0, wiz.RGBCW{W: 128}},
		{255, 255, 0, wiz.RGBCW{R: 255, G: 255}},
	} {
		got, err := wiz.RGBToRGBWPywizlight(c.r, c.g, c.b)
		if err != nil {
			t.Fatal(err)
		}
		if !near(got.R, c.want.R) || !near(got.G, c.want.G) || !near(got.B, c.want.B) || got.W != c.want.W {
			t.Errorf("RGBToRGBWPywizlight(%d, %d, %d) = %+v, want %+v", c.r, c.g, c.b, got, c.want)
		}
	}
	if _, err := wiz.RGBToRGBWPywizlight(256, 0, 0); err == nil {
		t.Error("256 accepted")
	}
	p := wiz.RGBCW{R: 1, G: 2, B: 3, C: 4, W: 5}.RGBWParams()
	if _, hasC := p["c"]; hasC || p["w"] != 5 || len(p) != 4 {
		t.Errorf("RGBWParams %v", p)
	}
}

func TestPulse(t *testing.T) {
	if _, err := wiz.PulseParams(0, time.Second); err == nil {
		t.Error("delta 0 accepted")
	}
	if _, err := wiz.PulseParams(101, time.Second); err == nil {
		t.Error("delta 101 accepted")
	}
	if _, err := wiz.PulseParams(-15, 0); err == nil {
		t.Error("duration 0 accepted")
	}
	p, err := wiz.PulseParams(-15, 300*time.Millisecond)
	if err != nil || p["delta"] != -15 || p["duration"] != int64(300) {
		t.Fatalf("PulseParams %v %v", p, err)
	}
	l := newFakeLight(t, "a8bb50e4c101")
	if err := wiz.Pulse(context.Background(), l.addr(), 20, 2*time.Second, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := l.lastParams(wiz.MethodPulse); got["delta"] != 20.0 || got["duration"] != 2000.0 || l.count(wiz.MethodPulse) != 1 {
		t.Errorf("pulse sent %v (%d times)", got, l.count(wiz.MethodPulse))
	}
}

func TestHeads(t *testing.T) {
	if _, err := wiz.HeadParams(0, nil); err == nil {
		t.Error("head 0 accepted")
	}
	if _, err := wiz.HeadParams(3, nil); err == nil {
		t.Error("head 3 accepted")
	}
	in := map[string]any{"state": true}
	p, err := wiz.HeadParams(wiz.HeadSecond, in)
	if err != nil || p["devices"] != 2 || p["state"] != true {
		t.Fatalf("HeadParams %v %v", p, err)
	}
	if _, changed := in["devices"]; changed {
		t.Error("HeadParams changed its argument")
	}

	ctx := context.Background()
	l := newFakeLight(t, "a8bb50e4c107")
	l.set(func(l *fakeLight) { l.heads = true })
	for _, head := range []int{wiz.HeadFirst, wiz.HeadSecond} {
		pl, err := wiz.GetHeadPilot(ctx, l.addr(), head, 3*time.Second)
		if err != nil || pl.Head() != head {
			t.Fatalf("head %d: %+v %v", head, pl, err)
		}
		if sel := l.lastParams(wiz.MethodGetPilot)["devices"]; sel != float64(head-1) {
			t.Errorf("head %d asked with devices %v, want %d", head, sel, head-1)
		}
	}
	// A lamp without a state per head answers untagged.
	l.set(func(l *fakeLight) { l.heads = false })
	if _, err := wiz.GetHeadPilot(ctx, l.addr(), wiz.HeadFirst, 3*time.Second); !errors.Is(err, wiz.ErrNoHeadState) {
		t.Errorf("untagged answer: %v", err)
	}
	if _, err := wiz.GetHeadPilot(ctx, l.addr(), 3, time.Second); err == nil || errors.Is(err, wiz.ErrNoHeadState) {
		t.Errorf("head 3: %v", err)
	}
}

// A system configuration whose homeId or roomId is not a number (as in
// pywizlight's fake bulbs, whose captures have them redacted) is read
// without an error; those fields stay nil.
func TestParseSystemConfigTolerant(t *testing.T) {
	c, err := wiz.ParseSystemConfig([]byte(`{"mac":"d8a0119906b7","homeId":"**REDACTED**","roomId":"**REDACTED**","typeId":0,"moduleName":"ESP25_SOCKET_01","fwVersion":"1.26.2","groupId":0}`))
	if err != nil || c.MAC != "d8a0119906b7" || c.ModuleName != "ESP25_SOCKET_01" || c.HomeID != nil || c.RoomID != nil || c.TypeID == nil || *c.TypeID != 0 {
		t.Errorf("%+v %v", c, err)
	}
	c, err = wiz.ParseSystemConfig([]byte(`{"mac":"a8bb50e4c101","homeId":4242,"roomId":7}`))
	if err != nil || *c.HomeID != 4242 || *c.RoomID != 7 || c.TypeID != nil {
		t.Errorf("%+v %v", c, err)
	}
	if _, err := wiz.ParseSystemConfig([]byte(`not json`)); err == nil {
		t.Error("broken JSON accepted")
	}
}

func TestPilotRhythmAndHead(t *testing.T) {
	p, err := wiz.ParsePilot([]byte(`{"mac":"a8bb50e4c101","state":true,"sceneId":14,"schdPsetId":9,"devices":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.EffectiveSceneID() != wiz.SceneRhythm || p.Head() != wiz.HeadSecond {
		t.Errorf("scene %d head %d", p.EffectiveSceneID(), p.Head())
	}
	p, _ = wiz.ParsePilot([]byte(`{"sceneId":14,"devices":5}`))
	if p.EffectiveSceneID() != 14 || p.Head() != 0 {
		t.Errorf("scene %d head %d", p.EffectiveSceneID(), p.Head())
	}
	if (wiz.Pilot{}).EffectiveSceneID() != 0 {
		t.Error("empty pilot has a scene")
	}
}
