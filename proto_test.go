package wiz

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Covers: SIG-IPE-4341
func TestWiZModuleClasses(t *testing.T) {
	for module, want := range map[string]Class{
		"ESP01_SHRGB1C_31": ClassRGB, "ESP03_SHRGB3_01ABI": ClassRGB, "ESP56_SHTW3_01": ClassTW, "ESP05_SHDW9_01": ClassDW,
		"ESP06_SHDW1_01": ClassDW, "ESP10_SOCKET_06": ClassSocket, "ESP25_SOCKET_01": ClassSocket,
	} {
		if c, ok := ClassOf(module); !ok || c != want {
			t.Errorf("%s: %s %v, want %s", module, c, ok, want)
		}
	}
	for _, module := range []string{"", "LIGHT", "ESP03_FANDIMS_01", "ESP20_DDTW1_01", "ESP99_XYZ_01", "OTHER_SHRGB_01"} {
		if c, ok := ClassOf(module); ok || c != ClassUnknown {
			t.Errorf("%q: %s %v, want unknown", module, c, ok)
		}
	}
}

// Covers: SIG-IPE-4344
func TestWiZRequestResends(t *testing.T) {
	// A device that loses the first two datagrams still answers within the
	// timeout, and answers to other methods are ignored.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var got atomic.Int32
	go func() {
		buf := make([]byte, 1024)
		for {
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			var m Message
			_ = json.Unmarshal(buf[:n], &m)
			if got.Add(1) <= 2 {
				continue
			}
			_, _ = conn.WriteToUDP([]byte(`{"method":"getSystemConfig","env":"pro","result":{}}`), from)
			_, _ = conn.WriteToUDP([]byte(`{"method":"`+m.Method+`","env":"pro","result":{"mac":"a8bb50e4c1ff","state":true}}`), from)
		}
	}()
	start := time.Now()
	res, err := Request(context.Background(), conn.LocalAddr().String(), MethodGetPilot, nil, 8*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := ParsePilot(res)
	if p.MAC != "a8bb50e4c1ff" || p.State == nil || !*p.State || got.Load() != 3 || time.Since(start) < 2*time.Second {
		t.Errorf("pilot %+v after %d sends in %s", p, got.Load(), time.Since(start))
	}
	// An error object is the device's refusal.
	go func() {}()
	srv, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer srv.Close()
	go func() {
		buf := make([]byte, 1024)
		n, from, err := srv.ReadFromUDP(buf)
		if err == nil && n > 0 {
			_, _ = srv.WriteToUDP([]byte(`{"method":"setPilot","env":"pro","error":{"code":-32602,"message":"Invalid params"}}`), from)
		}
	}()
	if _, err := Request(context.Background(), srv.LocalAddr().String(), MethodSetPilot, map[string]any{"dimming": 5}, 2*time.Second); !errors.Is(err, ErrDevice) {
		t.Errorf("refusal = %v", err)
	}
	// Silence ends at the timeout.
	quiet, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer quiet.Close()
	if _, err := Request(context.Background(), quiet.LocalAddr().String(), MethodGetPilot, nil, 1200*time.Millisecond); err == nil || !strings.Contains(err.Error(), "no answer") {
		t.Errorf("silence = %v", err)
	}
}

// Covers: SIG-IPE-4342
func TestWiZScenes(t *testing.T) {
	for name, id := range map[string]int{"Ocean": 1, "romance": 2, "Pastel colors": 8, "wake up": 9, "Wake-up": 9, "Warm white": 11,
		"Night light": 14, "Focus": 15, "Relax": 16, "TV time": 18, "Plantgrowth": 19, "Deepdive": 23, "Club": 26, "Christmas": 27,
		"Candlelight": 29, "Golden white": 30, "Steampunk": 32, "Diwali": 33, "Rhythm": 1000, "12": 12} {
		if got, ok := SceneID(name); !ok || got != id {
			t.Errorf("SceneID(%q) = %d %v, want %d", name, got, ok, id)
		}
	}
	if _, ok := SceneID("Moonwalk"); ok {
		t.Error("an unknown scene name was found")
	}
	if SceneName(0) != "" || SceneName(3) != "Sunset" || SceneName(77) != "scene 77" {
		t.Error("SceneName")
	}
	if n := len(ScenesFor(ClassRGB)); n != len(Scenes) {
		t.Errorf("RGB scenes = %d", n)
	}
	tw, dw := ScenesFor(ClassTW), ScenesFor(ClassDW)
	if !slices.Contains(tw, "Cozy") || slices.Contains(tw, "Ocean") || !slices.Contains(dw, "Night light") || slices.Contains(dw, "Cozy") || ScenesFor(ClassSocket) != nil {
		t.Errorf("tw %v dw %v", tw, dw)
	}
	if HasScene(ClassDW, 1) || !HasScene(ClassDW, 14) || !HasScene(ClassRGB, 1) || HasScene(ClassSocket, 14) {
		t.Error("HasScene")
	}
}
