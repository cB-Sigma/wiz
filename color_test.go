// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/cB-Sigma/wiz"
)

func TestHSToRGBCW(t *testing.T) {
	for _, tc := range []struct {
		h, s float64
		want wiz.RGBCW
	}{
		{0, 1, wiz.RGBCW{R: 255}},
		{120, 1, wiz.RGBCW{G: 255}},
		{240, 1, wiz.RGBCW{B: 255}},
		{60, 1, wiz.RGBCW{R: 255, G: 255}},
		{-120, 1, wiz.RGBCW{B: 255}},
		{480, 1, wiz.RGBCW{G: 255}},
		{300, 0.5, wiz.RGBCW{R: 128, B: 128, C: 128, W: 128}},
		{17, 0, wiz.RGBCW{C: 255, W: 255}},
	} {
		got, err := wiz.HSToRGBCW(tc.h, tc.s)
		if err != nil || got != tc.want {
			t.Errorf("HSToRGBCW(%v, %v) = %+v, %v; want %+v", tc.h, tc.s, got, err, tc.want)
		}
	}
	for _, bad := range [][2]float64{{0, -0.1}, {0, 1.1}, {math.NaN(), 1}, {math.Inf(1), 1}, {0, math.NaN()}} {
		if _, err := wiz.HSToRGBCW(bad[0], bad[1]); err == nil {
			t.Errorf("HSToRGBCW(%v, %v) accepted", bad[0], bad[1])
		}
	}
	p := wiz.RGBCW{R: 1, G: 2, B: 3, C: 4, W: 5}.Params()
	if p["r"] != 1 || p["g"] != 2 || p["b"] != 3 || p["c"] != 4 || p["w"] != 5 {
		t.Errorf("Params = %v", p)
	}
}

func TestRGBToRGBCW(t *testing.T) {
	got, err := wiz.RGBToRGBCW(255, 128, 128)
	if err != nil || got != (wiz.RGBCW{R: 127, C: 128, W: 128}) {
		t.Errorf("RGBToRGBCW = %+v, %v", got, err)
	}
	if got, _ := wiz.RGBToRGBCW(10, 200, 30); got != (wiz.RGBCW{G: 190, B: 20, C: 10, W: 10}) {
		t.Errorf("RGBToRGBCW = %+v", got)
	}
	if _, err := wiz.RGBToRGBCW(256, 0, 0); err == nil {
		t.Error("256 accepted")
	}
}

// A hue and saturation from an app become the five channels of setPilot.
func ExampleHSToRGBCW() {
	c, _ := wiz.HSToRGBCW(30, 0.8) // orange, a little washed out
	fmt.Printf("%+v\n", c)
	fmt.Println(len(c.Params()), "params")
	// Output:
	// {R:204 G:102 B:0 C:51 W:51}
	// 5 params
}
