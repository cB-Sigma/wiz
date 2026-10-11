// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"fmt"
	"math"
)

// pywizlight's colour mapping, as an option.
//
// pywizlight (0.6.6) maps a hue and saturation to the channels differently
// from HSToRGBCW. Its behaviour, read from pywizlight's source and written
// here in our own code, is:
//
//   - red, green and blue sit 120 degrees apart on a colour wheel (red at
//     0). A hue between two of them is split between those two in the
//     proportions that add up to the hue's direction, scaled so that the
//     larger is full; a hue on one of them drives that one alone.
//   - From saturation 0.5 to 1 that colour stays full and the warm white
//     LED fades from 128 to 0; below 0.5 the warm white LED stays at 128
//     and the colour fades towards black at saturation 0.
//   - The cool white LED is not used; channels are cut to whole numbers,
//     not rounded.
//   - An RGB colour is first turned into a hue and a saturation on the same
//     wheel: the length of the sum of the three channels' directions
//     (each 0 to 1) is the saturation.
//
// The results agree with pywizlight's to within one step per channel
// (rounding of the arithmetic can differ at the cut); the interop module
// checks this against pywizlight itself.

// pyEpsilon is the small value under which pywizlight treats a
// saturation, or a vector length squared, as zero.
const pyEpsilon = 1e-5

// pyWhiteMax is the largest warm-white value pywizlight sends.
const pyWhiteMax = 128

// primaries are the unit directions of red, green and blue on the wheel.
var primaries = [3][2]float64{
	{1, 0},
	{math.Cos(2 * math.Pi / 3), math.Sin(2 * math.Pi / 3)},
	{math.Cos(4 * math.Pi / 3), math.Sin(4 * math.Pi / 3)},
}

// HSToRGBWPywizlight converts a hue (degrees, any value) and a saturation
// (0 to 1) to the channels the way pywizlight does (see above): r, g, b
// and the warm white LED (W); C is always 0. Send it with RGBWParams to
// leave out c, as pywizlight does.
func HSToRGBWPywizlight(hue, sat float64) (RGBCW, error) {
	if math.IsNaN(hue) || math.IsInf(hue, 0) || math.IsNaN(sat) || sat < 0 || sat > 1 {
		return RGBCW{}, fmt.Errorf("wiz: hue %v and saturation %v (0 to 1) do not make a colour", hue, sat)
	}
	a := math.Mod(hue, 360) * math.Pi / 180
	return wheelToRGBW([2]float64{math.Cos(a), math.Sin(a)}, sat), nil
}

// RGBToRGBWPywizlight converts an RGB colour (each 0 to 255) to the
// channels the way pywizlight does (see above): r, g, b and the warm
// white LED (W); C is always 0.
func RGBToRGBWPywizlight(r, g, b int) (RGBCW, error) {
	for _, v := range []int{r, g, b} {
		if v < 0 || v > 255 {
			return RGBCW{}, fmt.Errorf("wiz: r, g, b %d, %d, %d are not all 0 to 255", r, g, b)
		}
	}
	var v [2]float64
	for i, c := range []int{r, g, b} {
		f := float64(c) / 255
		v[0] += primaries[i][0] * f
		v[1] += primaries[i][1] * f
	}
	sat := 0.0
	if sq := v[0]*v[0] + v[1]*v[1]; sq > pyEpsilon {
		sat = math.Sqrt(sq)
		v[0], v[1] = v[0]/sat, v[1]/sat
	}
	return wheelToRGBW(v, sat), nil
}

// wheelToRGBW maps a unit direction on the wheel and a saturation to the
// channels.
func wheelToRGBW(u [2]float64, sat float64) RGBCW {
	var rgb [3]float64
	if sat > pyEpsilon {
		// The primaries less than 120 degrees from u (one or two).
		limit := math.Cos(2*math.Pi/3 - pyEpsilon)
		var near []int
		for i, p := range primaries {
			if u[0]*p[0]+u[1]*p[1] > limit {
				near = append(near, i)
			}
		}
		switch len(near) {
		case 1:
			rgb[near[0]] = 1
		case 2:
			// Solve u = x*pa + y*pb for x and y.
			pa, pb := primaries[near[0]], primaries[near[1]]
			det := pa[0]*pb[1] - pa[1]*pb[0]
			x := (u[0]*pb[1] - u[1]*pb[0]) / det
			y := (pa[0]*u[1] - pa[1]*u[0]) / det
			m := math.Max(x, y)
			rgb[near[0]] = math.Min(x/m, 1)
			rgb[near[1]] = math.Min(y/m, 1)
		}
	}
	white := 1.0
	if sat >= 0.5 {
		white = 1 - (sat-0.5)*2
	} else {
		for i := range rgb {
			rgb[i] *= sat * 2
		}
	}
	return RGBCW{
		R: int(rgb[0] * 255), G: int(rgb[1] * 255), B: int(rgb[2] * 255),
		W: int(math.Max(0, white*pyWhiteMax)),
	}
}

// RGBWParams returns setPilot params with r, g, b and w only, leaving the
// cool white LED as it is; this is what pywizlight sends for a colour.
func (x RGBCW) RGBWParams() map[string]any {
	return map[string]any{"r": x.R, "g": x.G, "b": x.B, "w": x.W}
}
