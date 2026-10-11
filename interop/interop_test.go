// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package interop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cB-Sigma/wiz"
)

// The peer: pywizlight at this version, in this image.
const (
	image             = "python:3.12.10-slim"
	pywizlightPackage = "pywizlight==0.6.6"
	basePort          = 40000 // the relays' ports in the container
)

// The fake bulbs started, as module@firmware keys of pywizlight's fake.
var bulbs = []string{
	"ESP01_SHRGB_03@1.25.0",
	"ESP10_SOCKET_06@1.25.0",
	"ESP25_SOCKET_01@1.26.2",
	"ESP25_SOCKET_01@1.26.1",
	"ESP03_FANDIMS_31@1.31.32",
	"1.8.0-MISSING-TYPEID-0@1.8.0",
	"ESP20_DHRGB_01@1.35.0",
	"BROKEN_JSON@1.0.0",
	"ESP20_SHRGB_01BT@1.23.70",
	"ESP21_SHTW_01@1.25.0",
	"ESP06_SHDW9_01@1.11.7",
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

// pyBulb is what the script reports for a fake bulb, with pywizlight's own
// reading of it.
type pyBulb struct {
	Index    int    `json:"bulb"`
	Module   string `json:"module"`
	FW       string `json:"fw"`
	BulbType struct {
		Error       string `json:"error"`
		BulbType    string `json:"bulb_type"`
		KelvinRange *struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"kelvin_range"`
		FanSpeedRange *int `json:"fan_speed_range"`
		Features      struct {
			DualHead bool `json:"dual_head"`
			Fan      bool `json:"fan"`
			Color    bool `json:"color"`
		} `json:"features"`
	} `json:"bulbtype"`
	addr string
}

// startFakes runs the fake bulbs in a container and returns them by
// module@firmware, each with the host address that reaches it.
func startFakes(t *testing.T) map[string]*pyBulb {
	t.Helper()
	if os.Getenv("SIGMA_WIZ_INTEROP") != "1" {
		t.Skip("runs pywizlight's fake bulbs in Docker; set SIGMA_WIZ_INTEROP=1 to run")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not on PATH")
	}
	td, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("sigma-wiz-fakebulb-%d", os.Getpid())
	args := []string{"run", "-d", "--name", name, "-v", td + ":/sigma:ro"}
	hostPorts := make([]int, len(bulbs))
	for i := range bulbs {
		hostPorts[i] = freeUDPPort(t)
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d/udp", hostPorts[i], basePort+i))
	}
	args = append(args, image, "sleep", "3600")
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("docker run: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "container", "rm", "-f", name).Run() })
	if out, err := exec.Command("docker", "exec", name, "pip", "install", "--no-cache-dir",
		"--disable-pip-version-check", "--root-user-action=ignore", pywizlightPackage).CombinedOutput(); err != nil {
		t.Fatalf("pip install %s: %v: %s", pywizlightPackage, err, out)
	}
	cmd := exec.Command("docker", append([]string{"exec", "-i", name, "python", "-I", "/sigma/fake_bulbs.py", "--base", fmt.Sprint(basePort)}, bulbs...)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = io.WriteString(stdin, "quit\n")
		_ = stdin.Close()
		_ = cmd.Wait()
	})
	found := map[string]*pyBulb{}
	ready := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			txt := sc.Text()
			switch {
			case strings.HasPrefix(txt, `{"pywizlight"`):
				t.Logf("peer %s", txt)
			case strings.HasPrefix(txt, `{"bulb"`):
				var b pyBulb
				if err := json.Unmarshal([]byte(txt), &b); err != nil {
					ready <- err
					return
				}
				b.addr = fmt.Sprintf("127.0.0.1:%d", hostPorts[b.Index])
				found[b.Module+"@"+b.FW] = &b
				t.Logf("fake bulb %s@%s: pywizlight reads %+v", b.Module, b.FW, b.BulbType)
			case strings.HasPrefix(txt, `{"ready"`):
				ready <- nil
				// Keep draining the fake bulbs' own request log.
				for sc.Scan() {
				}
				return
			}
		}
		ready <- errors.New("the fake bulb script ended before it was ready")
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Minute):
		t.Fatal("the fake bulbs did not start")
	}
	return found
}

const timeout = 5 * time.Second

func TestClientAgainstPywizlightFakeBulbs(t *testing.T) {
	fakes := startFakes(t)
	ctx := context.Background()

	for key, b := range fakes {
		if b.Module == "BROKEN_JSON" {
			continue
		}
		t.Run(key, func(t *testing.T) {
			raw, err := wiz.Request(ctx, b.addr, wiz.MethodGetSystemConfig, nil, timeout)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := wiz.ParseSystemConfig(raw)
			if err != nil || cfg.MAC == "" || cfg.FwVersion == "" {
				t.Fatalf("system config %+v %v (%s)", cfg, err, raw)
			}
			class, ok := wiz.ClassOfConfig(cfg)
			if b.BulbType.Error != "" {
				// pywizlight could not read this bulb (both configuration
				// methods answer "method not found"); only our own reading
				// is checked.
				t.Logf("pywizlight: %s; this module reads class %s", b.BulbType.Error, class)
				if !ok {
					t.Errorf("class of %s unknown", cfg.ModuleName)
				}
				return
			}
			pyClass := map[string]wiz.Class{"RGB": wiz.ClassRGB, "TW": wiz.ClassTW, "DW": wiz.ClassDW, "SOCKET": wiz.ClassSocket,
				"FANDIM": wiz.ClassFanDim, "FANTW": wiz.ClassFanTW}[b.BulbType.BulbType]
			if !ok || class != pyClass {
				t.Errorf("class %s %v, pywizlight reads %s", class, ok, b.BulbType.BulbType)
			}
			f := wiz.FeaturesOf(cfg.ModuleName)
			if cfg.ModuleName != "" && (f.DualHead != b.BulbType.Features.DualHead || f.Fan != b.BulbType.Features.Fan || f.Color != b.BulbType.Features.Color) {
				t.Errorf("features %+v, pywizlight reads %+v", f, b.BulbType.Features)
			}

			model, err := wiz.GetModelConfig(ctx, b.addr, timeout)
			var de *wiz.DeviceError
			if err != nil && !(errors.As(err, &de) && de.Code == wiz.CodeMethodNotFound) {
				t.Fatalf("model config: %v", err)
			}
			user, err := wiz.GetUserConfig(ctx, b.addr, timeout)
			if err != nil && !(errors.As(err, &de) && de.Code == wiz.CodeMethodNotFound) {
				t.Fatalf("user config: %v", err)
			}
			lo, hi, ok := wiz.KelvinRange(model, user)
			if kr := b.BulbType.KelvinRange; kr != nil {
				if !ok || lo != kr.Min || hi != kr.Max {
					t.Errorf("kelvin %d..%d %v, pywizlight reads %d..%d", lo, hi, ok, kr.Min, kr.Max)
				}
			} else if ok {
				t.Errorf("kelvin %d..%d, pywizlight reads none", lo, hi)
			}
			if b.BulbType.Features.Fan {
				want := wiz.DefaultFanSpeedMax
				if n := b.BulbType.FanSpeedRange; n != nil && *n > 0 {
					want = *n
				}
				if got := wiz.FanSpeedMax(model, user); got != want {
					t.Errorf("fan speeds %d, pywizlight reads %v", got, b.BulbType.FanSpeedRange)
				}
			}

			raw, err = wiz.Request(ctx, b.addr, wiz.MethodGetPilot, nil, timeout)
			if err != nil {
				t.Fatal(err)
			}
			p, err := wiz.ParsePilot(raw)
			if err != nil || p.MAC != "abcabcabcabc" || p.State == nil || *p.Dimming != 13 {
				t.Errorf("pilot %+v %v", p, err)
			}
		})
	}

	t.Run("setPilot", func(t *testing.T) {
		b := fakes["ESP01_SHRGB_03@1.25.0"]
		colour, _ := wiz.HSToRGBCW(200, 0.8)
		params := colour.Params()
		params["dimming"] = 40
		params["state"] = true
		if _, err := wiz.Request(ctx, b.addr, wiz.MethodSetPilot, params, timeout); err != nil {
			t.Fatal(err)
		}
		raw, _ := wiz.Request(ctx, b.addr, wiz.MethodGetPilot, nil, timeout)
		p, _ := wiz.ParsePilot(raw)
		if !*p.State || *p.Dimming != 40 || int(*p.R) != colour.R || int(*p.C) != colour.C {
			t.Errorf("pilot after setPilot %s", raw)
		}
	})

	t.Run("dual head ratio", func(t *testing.T) {
		b := fakes["ESP20_DHRGB_01@1.35.0"]
		params, _ := wiz.RatioParams(30)
		if _, err := wiz.Request(ctx, b.addr, wiz.MethodSetPilot, params, timeout); err != nil {
			t.Fatal(err)
		}
		raw, _ := wiz.Request(ctx, b.addr, wiz.MethodGetPilot, nil, timeout)
		if p, _ := wiz.ParsePilot(raw); p.Ratio == nil || *p.Ratio != 30 {
			t.Errorf("pilot after ratio %s", raw)
		}
	})

	t.Run("fan", func(t *testing.T) {
		b := fakes["ESP03_FANDIMS_31@1.31.32"]
		model, _ := wiz.GetModelConfig(ctx, b.addr, timeout)
		on := true
		params, err := wiz.FanParams{On: &on, Mode: wiz.FanModeNormal, Speed: 6}.Params(wiz.FanSpeedMax(model, wiz.UserConfig{}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := wiz.Request(ctx, b.addr, wiz.MethodSetPilot, params, timeout); err != nil {
			t.Fatal(err)
		}
		raw, _ := wiz.Request(ctx, b.addr, wiz.MethodGetPilot, nil, timeout)
		if p, _ := wiz.ParsePilot(raw); p.FanSpeed == nil || *p.FanSpeed != 6 || *p.FanState != 1 {
			t.Errorf("pilot after fan %s", raw)
		}
	})

	t.Run("power", func(t *testing.T) {
		mw, err := wiz.GetPower(ctx, fakes["ESP25_SOCKET_01@1.26.2"].addr, timeout)
		if err != nil || mw != 1065385 {
			t.Errorf("metering plug %v %v", mw, err)
		}
		if _, err := wiz.GetPower(ctx, fakes["ESP25_SOCKET_01@1.26.1"].addr, timeout); !errors.Is(err, wiz.ErrNoReading) {
			t.Errorf("answer without a result: %v", err)
		}
		var de *wiz.DeviceError
		if _, err := wiz.GetPower(ctx, fakes["ESP10_SOCKET_06@1.25.0"].addr, timeout); !errors.As(err, &de) || de.Code != wiz.CodeMethodNotFound {
			t.Errorf("plug without metering: %v", err)
		}
	})

	t.Run("old firmware without a module name", func(t *testing.T) {
		raw, _ := wiz.Request(ctx, fakes["1.8.0-MISSING-TYPEID-0@1.8.0"].addr, wiz.MethodGetSystemConfig, nil, timeout)
		cfg, _ := wiz.ParseSystemConfig(raw)
		if cfg.ModuleName != "" || cfg.TypeID == nil || cfg.WhiteChannels == nil || *cfg.WhiteChannels != 1 || *cfg.WhiteToColorRatio != 20 {
			t.Errorf("system config %+v", cfg)
		}
	})

	t.Run("registration and discovery answer", func(t *testing.T) {
		b := fakes["ESP01_SHRGB_03@1.25.0"]
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		dev, _ := net.ResolveUDPAddr("udp4", b.addr)
		if err := wiz.Register(conn, dev, "02aabbccddee", "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
		// The fake answers a stray getModelConfig error first, then the
		// registration.
		var got wiz.Message
		buf := make([]byte, 4096)
		for got.Method != wiz.MethodRegistration {
			_ = conn.SetReadDeadline(time.Now().Add(timeout))
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				t.Fatal(err)
			}
			_ = json.Unmarshal(buf[:n], &got)
		}
		if !strings.Contains(string(got.Result), `"success": true`) && !strings.Contains(string(got.Result), `"success":true`) {
			t.Errorf("registration answer %s", got.Result)
		}
		// Discover sent to the fake bulb's address finds it by its MAC.
		found, err := wiz.Discover(ctx, b.addr, "02aabbccddee", 1500*time.Millisecond)
		if err != nil || found["a8bb5006033d"] == "" {
			t.Errorf("discover %v %v", found, err)
		}
	})

	t.Run("broken JSON", func(t *testing.T) {
		b := fakes["BROKEN_JSON@1.0.0"]
		// The fake answers getSystemConfig with garbage: no answer counts.
		if _, err := wiz.Request(ctx, b.addr, wiz.MethodGetSystemConfig, nil, 1500*time.Millisecond); err == nil {
			t.Error("garbage taken as an answer")
		}
		if _, err := wiz.Request(ctx, b.addr, wiz.MethodGetPilot, nil, timeout); err != nil {
			t.Errorf("getPilot after garbage: %v", err)
		}
	})
}
