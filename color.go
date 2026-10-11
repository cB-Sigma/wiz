// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"fmt"
	"math"
)

// Colour conversion. WiZ colour lights take five channels in setPilot: r,
// g, b and the cool (c) and warm (w) white LEDs, each 0 to 255. Apps and
// home-automation systems usually hold a colour as hue and saturation, or
// as plain RGB. The conversions below are our own simple mapping, not a
// measured colour model and not pywizlight's: the coloured part goes to
// r, g and b, and the white part is driven on both white LEDs equally.
// Brightness is set separately with dimming.

// RGBCW is a setPilot colour.
type RGBCW struct {
	R, G, B, C, W int
}

// Params returns the setPilot params for the colour.
func (x RGBCW) Params() map[string]any {
	return map[string]any{"r": x.R, "g": x.G, "b": x.B, "c": x.C, "w": x.W}
}

// HSToRGBCW converts a hue (degrees, any value; taken modulo 360) and a
// saturation (0 to 1) to the five channels: at saturation 1 the pure hue
// on r, g and b with the white LEDs off; at 0 both white LEDs full and r,
// g, b off; in between, the hue scaled by the saturation and the white
// LEDs by the rest.
func HSToRGBCW(hue, sat float64) (RGBCW, error) {
	if math.IsNaN(hue) || math.IsInf(hue, 0) || math.IsNaN(sat) || sat < 0 || sat > 1 {
		return RGBCW{}, fmt.Errorf("wiz: hue %v and saturation %v (0 to 1) do not make a colour", hue, sat)
	}
	h := math.Mod(hue, 360)
	if h < 0 {
		h += 360
	}
	// The pure hue at full value (HSV with S = V = 1).
	x := 1 - math.Abs(math.Mod(h/60, 2)-1)
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = 1, x, 0
	case h < 120:
		r, g, b = x, 1, 0
	case h < 180:
		r, g, b = 0, 1, x
	case h < 240:
		r, g, b = 0, x, 1
	case h < 300:
		r, g, b = x, 0, 1
	default:
		r, g, b = 1, 0, x
	}
	white := ch((1 - sat) * 255)
	return RGBCW{R: ch(r * sat * 255), G: ch(g * sat * 255), B: ch(b * sat * 255), C: white, W: white}, nil
}

// RGBToRGBCW moves the grey part of an RGB colour (its smallest channel)
// to the white LEDs, which are brighter and whiter than mixing r, g and b:
// (255, 128, 128) becomes r 127, g 0, b 0 with both white LEDs at 128.
// Channels outside 0 to 255 are refused.
func RGBToRGBCW(r, g, b int) (RGBCW, error) {
	for _, v := range []int{r, g, b} {
		if v < 0 || v > 255 {
			return RGBCW{}, fmt.Errorf("wiz: r, g, b %d, %d, %d are not all 0 to 255", r, g, b)
		}
	}
	m := min(r, g, b)
	return RGBCW{R: r - m, G: g - m, B: b - m, C: m, W: m}, nil
}

func ch(v float64) int {
	return int(math.Max(0, math.Min(255, math.Round(v))))
}
