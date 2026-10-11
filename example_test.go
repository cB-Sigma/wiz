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
	for _, m := range []string{"ESP01_SHRGB1C_31", "ESP05_SHDW9_01", "ESP10_SOCKET_06", "ESP03_FANDIMS_01"} {
		c, ok := wiz.ClassOf(m)
		fmt.Println(m, c, ok)
	}
	// Output:
	// ESP01_SHRGB1C_31 rgb true
	// ESP05_SHDW9_01 dw true
	// ESP10_SOCKET_06 socket true
	// ESP03_FANDIMS_01 unknown false
}
