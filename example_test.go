// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

import (
	"context"
	"fmt"
	"time"

	"github.com/cB-Sigma/wiz"
)

// Find the lights, read one and switch it to a scene. It needs lights on
// the LAN, so it is not run; github.com/cB-Sigma/wiz-sim runs the same
// steps against simulated lights.
func Example() {
	ctx := context.Background()

	// The MAC is the one this client reports as its own.
	found, err := wiz.Discover(ctx, "255.255.255.255:38899", "02aabbccddee", time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	for mac, addr := range found {
		raw, err := wiz.Request(ctx, addr, wiz.MethodGetSystemConfig, nil, 3*time.Second)
		if err != nil {
			fmt.Println(mac, err)
			continue
		}
		cfg, _ := wiz.ParseSystemConfig(raw)
		class, _ := wiz.ClassOf(cfg.ModuleName)
		fmt.Println(mac, cfg.ModuleName, "is", class)

		if id, _ := wiz.SceneID("warm white"); wiz.HasScene(class, id) {
			_, err := wiz.Request(ctx, addr, wiz.MethodSetPilot, map[string]any{"sceneId": id}, 3*time.Second)
			fmt.Println("scene set:", err == nil)
		}
	}
}

func ExampleClassOf() {
	for _, m := range []string{"ESP01_SHRGB1C_31", "ESP05_SHDW9_01", "ESP10_SOCKET_06", "ESP03_FANDIMS_31", "ESP20_DDTW1_01", "ESP99_XYZ_01"} {
		c, ok := wiz.ClassOf(m)
		fmt.Println(m, c, ok)
	}
	// Output:
	// ESP01_SHRGB1C_31 rgb true
	// ESP05_SHDW9_01 dw true
	// ESP10_SOCKET_06 socket true
	// ESP03_FANDIMS_31 fandim true
	// ESP20_DDTW1_01 fantw true
	// ESP99_XYZ_01 unknown false
}

func ExampleFeaturesOf() {
	f := wiz.FeaturesOf("ESP03_FANDIMS_31")
	fmt.Println("fan:", f.Fan, "breeze:", f.FanBreeze, "reverse:", f.FanReverse, "colour temperature:", f.ColorTemp)
	fmt.Println("dual head:", wiz.FeaturesOf("ESP20_DHRGB_01").DualHead)
	// Output:
	// fan: true breeze: true reverse: true colour temperature: false
	// dual head: true
}

// Switch a ceiling fan on at speed 3 of the speeds it reports; its light
// is not touched.
func ExampleFanParams() {
	ctx := context.Background()
	addr := "192.168.1.60:38899" // a fan module

	model, _ := wiz.GetModelConfig(ctx, addr, 3*time.Second) // zero when the firmware has none
	user, _ := wiz.GetUserConfig(ctx, addr, 3*time.Second)
	on := true
	params, err := wiz.FanParams{On: &on, Mode: wiz.FanModeNormal, Speed: 3}.Params(wiz.FanSpeedMax(model, user))
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = wiz.Request(ctx, addr, wiz.MethodSetPilot, params, 3*time.Second)
	fmt.Println("fan on:", err == nil)
}

func ExampleKelvinRange() {
	model, _ := wiz.ParseModelConfig([]byte(`{"cctRange":[2200,2700,6500,6500],"fanSpeed":0}`))
	min, max, ok := wiz.KelvinRange(model, wiz.UserConfig{})
	fmt.Println(min, max, ok)
	min, max, ok = wiz.KelvinRange(wiz.ModelConfig{}, wiz.UserConfig{})
	fmt.Println(min, max, ok)
	// Output:
	// 2200 6500 true
	// 2200 6500 false
}
