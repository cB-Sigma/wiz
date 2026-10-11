// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cB-Sigma/wiz"
)

// fakeLight is a minimal light on the loopback interface for the
// library's own tests: it answers discovery, registration, getPilot and
// setPilot on its API port and pushes from a second socket. The full
// simulator, with every module class and the push timing of real lights,
// is github.com/cB-Sigma/wiz-sim.
type fakeLight struct {
	t    *testing.T
	mac  string
	api  *net.UDPConn
	push *net.UDPConn

	mu   sync.Mutex
	on   bool
	regs []map[string]any
	acks int
	// lastAck is the last syncPilot acknowledgement received.
	lastAck map[string]any
	// got counts requests by method; params keeps each method's last params.
	got    map[string]int
	params map[string]map[string]any
	// noPower makes getPower answer "method not found"; silent lists
	// methods the light never answers.
	noPower bool
	silent  map[string]bool
	// heads makes getPilot with a devices selector (0 or 1) answer tagged
	// with the head (1 or 2), as a dual-head lamp with a state per head.
	heads bool
}

func newFakeLight(t *testing.T, mac string) *fakeLight {
	t.Helper()
	api, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	push, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	l := &fakeLight{t: t, mac: mac, api: api, push: push, on: true, got: map[string]int{}, params: map[string]map[string]any{}, silent: map[string]bool{}}
	t.Cleanup(func() { _ = api.Close(); _ = push.Close() })
	go l.serveAPI()
	go l.servePushAcks()
	return l
}

func (l *fakeLight) addr() string { return l.api.LocalAddr().String() }

func (l *fakeLight) serveAPI() {
	buf := make([]byte, 4096)
	for {
		n, from, err := l.api.ReadFromUDP(buf)
		if err != nil {
			return
		}
		var m struct {
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if json.Unmarshal(buf[:n], &m) != nil {
			continue
		}
		var result any
		var refusal map[string]any
		l.mu.Lock()
		l.got[m.Method]++
		l.params[m.Method] = m.Params
		if l.silent[m.Method] {
			l.mu.Unlock()
			continue
		}
		switch m.Method {
		case wiz.MethodGetModelConfig:
			result = map[string]any{"cctRange": []int{2200, 2700, 6500, 6500}, "fanSpeed": 0, "wcr": 20, "nowc": 2}
		case wiz.MethodGetUserConfig:
			result = map[string]any{"fadeIn": 450, "extRange": []int{2700, 6500}, "po": false}
		case wiz.MethodGetWifiConfig:
			result = "opaque"
		case wiz.MethodGetPower:
			if l.noPower {
				refusal = map[string]any{"code": -32601, "message": "Method not found"}
			} else {
				result = map[string]any{"power": 61500}
			}
		case wiz.MethodSetUserConfig, wiz.MethodSetSystemConfig, wiz.MethodSetModelConfig, wiz.MethodReboot, wiz.MethodReset, wiz.MethodUpdateOta, wiz.MethodSetState, wiz.MethodPulse:
			result = map[string]any{"success": true}
		case wiz.MethodRegistration:
			if m.Params["register"] == true {
				l.regs = append(l.regs, m.Params)
			}
			result = map[string]any{"mac": l.mac, "success": true}
		case wiz.MethodGetPilot:
			p := map[string]any{"mac": l.mac, "state": l.on, "sceneId": 0}
			if sel, ok := m.Params["devices"].(float64); ok && l.heads {
				p["devices"] = sel + 1
			}
			result = p
		case wiz.MethodSetPilot:
			if s, ok := m.Params["state"].(bool); ok {
				l.on = s
			}
			result = map[string]any{"success": true}
		}
		l.mu.Unlock()
		if refusal != nil {
			b, _ := json.Marshal(map[string]any{"method": m.Method, "env": "pro", "error": refusal})
			_, _ = l.api.WriteToUDP(b, from)
			continue
		}
		if result == nil {
			continue
		}
		b, _ := json.Marshal(map[string]any{"method": m.Method, "env": "pro", "result": result})
		_, _ = l.api.WriteToUDP(b, from)
	}
}

func (l *fakeLight) servePushAcks() {
	buf := make([]byte, 4096)
	for {
		n, _, err := l.push.ReadFromUDP(buf)
		if err != nil {
			return
		}
		var m wiz.Message
		if json.Unmarshal(buf[:n], &m) == nil && m.Method == wiz.MethodSyncPilot && m.Result != nil {
			var raw map[string]any
			_ = json.Unmarshal(buf[:n], &raw)
			l.mu.Lock()
			l.acks++
			l.lastAck = raw
			l.mu.Unlock()
		}
	}
}

// sendPush sends a datagram from the push socket to the registered address.
func (l *fakeLight) sendPush(to string, msg map[string]any) {
	l.t.Helper()
	ua, err := net.ResolveUDPAddr("udp4", to)
	if err != nil {
		l.t.Fatal(err)
	}
	b, _ := json.Marshal(msg)
	if _, err := l.push.WriteToUDP(b, ua); err != nil {
		l.t.Fatal(err)
	}
}

func (l *fakeLight) snapshot() (on bool, regs []map[string]any, acks int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.on, append([]map[string]any(nil), l.regs...), l.acks
}

const testPhoneMAC = "02aabbccddee"

func TestDiscoverFakeLight(t *testing.T) {
	l := newFakeLight(t, "A8:BB:50:E4:C1:01")
	found, err := wiz.Discover(context.Background(), l.addr(), testPhoneMAC, 1200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found["a8bb50e4c101"] != l.addr() {
		t.Fatalf("found %v, want a8bb50e4c101 at %s", found, l.addr())
	}
	if _, regs, _ := l.snapshot(); len(regs) != 0 {
		t.Errorf("discovery registered for pushes: %v", regs)
	}
}

func TestRequestAndPushFakeLight(t *testing.T) {
	l := newFakeLight(t, "a8bb50e4c101")
	ctx := context.Background()

	raw, err := wiz.Request(ctx, l.addr(), wiz.MethodSetPilot, map[string]any{"state": false}, 3*time.Second)
	if err != nil {
		t.Fatalf("setPilot: %v (%s)", err, raw)
	}
	raw, err = wiz.Request(ctx, l.addr(), wiz.MethodGetPilot, nil, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p, err := wiz.ParsePilot(raw)
	if err != nil || p.State == nil || *p.State {
		t.Fatalf("getPilot %s: %+v %v", raw, p, err)
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	pushes := make(chan wiz.Push, 8)
	done := make(chan error, 1)
	go func() { done <- wiz.ServePush(conn, testPhoneMAC, func(p wiz.Push) { pushes <- p }) }()

	dev, _ := net.ResolveUDPAddr("udp4", l.addr())
	if got := wiz.LocalIPFor(dev); got != "127.0.0.1" {
		t.Errorf("LocalIPFor = %q", got)
	}
	if err := wiz.Register(conn, dev, testPhoneMAC, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { _, regs, _ := l.snapshot(); return len(regs) == 1 })
	if _, regs, _ := l.snapshot(); regs[0]["phoneMac"] != testPhoneMAC || regs[0]["phoneIp"] != "127.0.0.1" {
		t.Errorf("registration %v", regs[0])
	}

	// The registration's answer reaches the push socket and is skipped; a
	// syncPilot from the device's other port is acknowledged and handed on,
	// and so is a firstBeat. A syncPilot without params (an answer) and a
	// datagram without a method are skipped.
	to := conn.LocalAddr().String()
	l.sendPush(to, map[string]any{"method": "syncPilot"})
	l.sendPush(to, map[string]any{"other": true})
	l.sendPush(to, map[string]any{"method": wiz.MethodSyncPilot, "env": "pro", "params": map[string]any{"mac": "a8bb50e4c101", "state": true}})
	p1 := nextPush(t, pushes)
	if p1.Method != wiz.MethodSyncPilot || p1.MAC != "a8bb50e4c101" || p1.Pilot.State == nil || !*p1.Pilot.State {
		t.Errorf("push %+v", p1)
	}
	if got := wiz.PushedAddr(p1.From, l.addr()); got != l.addr() {
		t.Errorf("PushedAddr = %s, want %s", got, l.addr())
	}
	waitUntil(t, func() bool { _, _, acks := l.snapshot(); return acks >= 1 })
	l.sendPush(to, map[string]any{"method": wiz.MethodFirstBeat, "params": map[string]any{"mac": "A8BB50E4C101"}})
	if p2 := nextPush(t, pushes); p2.Method != wiz.MethodFirstBeat || p2.MAC != "a8bb50e4c101" {
		t.Errorf("first beat %+v", p2)
	}

	_ = conn.Close()
	if err := <-done; err == nil {
		t.Error("ServePush returned nil after the socket closed")
	}
}

func nextPush(t *testing.T, c chan wiz.Push) wiz.Push {
	t.Helper()
	select {
	case p := <-c:
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("no push")
	}
	return wiz.Push{}
}

func waitUntil(t *testing.T, f func() bool) {
	t.Helper()
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if f() {
			return
		}
	}
	t.Fatal("condition not met in 5s")
}
