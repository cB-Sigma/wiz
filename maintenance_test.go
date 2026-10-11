// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/cB-Sigma/wiz"
)

func (l *fakeLight) count(method string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.got[method]
}

func (l *fakeLight) lastParams(method string) map[string]any {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.params[method]
}

func (l *fakeLight) set(f func(*fakeLight)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f(l)
}

func TestConfigRequests(t *testing.T) {
	l := newFakeLight(t, "a8bb50e4c101")
	ctx := context.Background()

	m, err := wiz.GetModelConfig(ctx, l.addr(), 3*time.Second)
	if err != nil || !reflect.DeepEqual(m.CCTRange, []int{2200, 2700, 6500, 6500}) || *m.WCR != 20 {
		t.Fatalf("model config %+v %v", m, err)
	}
	u, err := wiz.GetUserConfig(ctx, l.addr(), 3*time.Second)
	if err != nil || *u.FadeIn != 450 || !reflect.DeepEqual(u.ExtRange, []int{2700, 6500}) {
		t.Fatalf("user config %+v %v", u, err)
	}
	if min, max, ok := wiz.KelvinRange(m, u); !ok || min != 2200 || max != 6500 {
		t.Errorf("kelvin %d..%d %v", min, max, ok)
	}
	if n := wiz.FanSpeedMax(m, u); n != wiz.DefaultFanSpeedMax {
		t.Errorf("fan speeds %d", n)
	}
	raw, err := wiz.GetWifiConfig(ctx, l.addr(), 3*time.Second)
	if err != nil || string(raw) != `"opaque"` {
		t.Errorf("wifi config %s %v", raw, err)
	}

	dim := 70
	if err := wiz.SetUserConfig(ctx, l.addr(), wiz.UserConfigParams{DftDim: &dim}, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if p := l.lastParams(wiz.MethodSetUserConfig); p["dftDim"] != 70.0 || p["userConfigTs"] != 0.0 {
		t.Errorf("setUserConfig params %v", p)
	}
	if err := wiz.SetUserConfig(ctx, l.addr(), wiz.UserConfigParams{}, time.Second); err == nil || l.count(wiz.MethodSetUserConfig) != 1 {
		t.Error("an empty setUserConfig was sent")
	}

	if _, err := wiz.SetSystemConfig(ctx, l.addr(), map[string]any{"fs": 1}, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := wiz.SetModelConfig(ctx, l.addr(), map[string]any{"confTs": 0}, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if l.lastParams(wiz.MethodSetSystemConfig)["fs"] != 1.0 || l.lastParams(wiz.MethodSetModelConfig)["confTs"] != 0.0 {
		t.Error("raw config params not sent as given")
	}
	for _, f := range []func() (json.RawMessage, error){
		func() (json.RawMessage, error) { return wiz.SetSystemConfig(ctx, l.addr(), nil, time.Second) },
		func() (json.RawMessage, error) {
			return wiz.SetModelConfig(ctx, l.addr(), map[string]any{}, time.Second)
		},
	} {
		if _, err := f(); err == nil {
			t.Error("empty raw config params were sent")
		}
	}
}

func TestPowerReading(t *testing.T) {
	l := newFakeLight(t, "a8bb50e4c104")
	ctx := context.Background()
	mw, err := wiz.GetPower(ctx, l.addr(), 3*time.Second)
	if err != nil || mw != 61500 {
		t.Fatalf("power %v %v", mw, err)
	}
	// A device without metering answers "method not found".
	l.set(func(l *fakeLight) { l.noPower = true })
	_, err = wiz.GetPower(ctx, l.addr(), 3*time.Second)
	var de *wiz.DeviceError
	if !errors.As(err, &de) || de.Code != wiz.CodeMethodNotFound || !errors.Is(err, wiz.ErrDevice) {
		t.Errorf("no metering = %v", err)
	}
}

func TestMaintenanceIsSentOnce(t *testing.T) {
	l := newFakeLight(t, "a8bb50e4c101")
	ctx := context.Background()
	if err := wiz.Reboot(ctx, l.addr(), 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := wiz.Reset(ctx, l.addr(), 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := wiz.UpdateFirmware(ctx, l.addr(), "", true, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if p := l.lastParams(wiz.MethodUpdateOta); p["fw"] != "default" || p["force"] != 1.0 {
		t.Errorf("updateOta params %v", p)
	}
	if _, err := wiz.UpdateFirmware(ctx, l.addr(), "1.30.0", false, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if p := l.lastParams(wiz.MethodUpdateOta); p["fw"] != "1.30.0" || p["force"] != 0.0 {
		t.Errorf("updateOta params %v", p)
	}
	if p := l.lastParams(wiz.MethodReboot); p == nil || len(p) != 0 {
		t.Errorf("reboot params %v, want an empty object", p)
	}
	// A light that does not answer gets the request once: a resend could
	// reach it after it restarted.
	l.set(func(l *fakeLight) { l.silent[wiz.MethodReboot] = true })
	if err := wiz.Reboot(ctx, l.addr(), 2*time.Second); err == nil {
		t.Error("a silent reboot reported success")
	}
	if n := l.count(wiz.MethodReboot); n != 2 {
		t.Errorf("reboot sent %d times in all, want 2", n)
	}
}

func TestPushKinds(t *testing.T) {
	l := newFakeLight(t, "a8bb50e4c101")
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	pushes := make(chan wiz.Push, 8)
	go func() { _ = wiz.ServePush(conn, testPhoneMAC, func(p wiz.Push) { pushes <- p }) }()
	to := conn.LocalAddr().String()

	// The acknowledgement echoes the push's id and env.
	l.sendPush(to, map[string]any{"method": wiz.MethodSyncPilot, "id": 42, "env": "dev", "params": map[string]any{"mac": "a8bb50e4c101", "state": true, "fanState": 1}})
	if p := nextPush(t, pushes); p.Pilot.FanState == nil || *p.Pilot.FanState != 1 || p.Params["fanState"] != 1.0 {
		t.Errorf("sync push %+v", p)
	}
	waitUntil(t, func() bool { _, _, n := l.snapshot(); return n >= 1 })
	l.mu.Lock()
	ack := l.lastAck
	l.mu.Unlock()
	if ack["id"] != 42.0 || ack["env"] != "dev" || ack["result"].(map[string]any)["mac"] != testPhoneMAC {
		t.Errorf("ack %v", ack)
	}

	l.sendPush(to, map[string]any{"method": wiz.MethodFirstBeat, "env": "pro", "params": map[string]any{"mac": "a8bb50e4c101", "fwVersion": "1.30.0", "matterVersion": "1.2", "productId": 17}})
	if p := nextPush(t, pushes); p.Method != wiz.MethodFirstBeat || p.FwVersion != "1.30.0" || p.MatterVersion != "1.2" || p.ProductID == nil || *p.ProductID != 17 {
		t.Errorf("first beat %+v", p)
	}

	// Firmware update progress; an answer to an updateOta sent from this
	// socket (no params) is not a push.
	l.sendPush(to, map[string]any{"method": wiz.MethodUpdateOta, "result": map[string]any{"success": true}})
	l.sendPush(to, map[string]any{"method": wiz.MethodUpdateOta, "env": "pro", "params": map[string]any{"updateStatus": 3}})
	if p := nextPush(t, pushes); p.Method != wiz.MethodUpdateOta || p.UpdateStatus == nil || *p.UpdateStatus != 3 || p.MAC != "" {
		t.Errorf("update push %+v", p)
	}
}
