// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

// Package wiz speaks the local UDP API of WiZ Wi-Fi lights, plugs and
// ceiling fans: JSON requests to port 38899 of the device with resends
// (Request), the light and fan state (Pilot, FanParams, RatioParams), the
// module classes (colour, tunable white, dimmable white, plug, fan with a
// dimmable or a tunable-white light) with their features (FeaturesOf) and
// built-in scenes, the system, model and user configuration
// (ParseSystemConfig, GetModelConfig, GetUserConfig, KelvinRange,
// FanSpeedMax, SetUserConfig), power readings (GetPower), maintenance
// (Reboot, Reset, UpdateFirmware), broadcast discovery (Discover), and
// push registration with the pushes devices send to port 38900
// (Register, ServePush, PushedAddr). It needs no cloud account.
//
// The local API is not published by its vendor; this package follows the
// behaviour public clients describe (see PROVENANCE.md). Where those
// descriptions stop (Wi-Fi setup of new devices, the meaning of some
// values), the package passes data through as sent and says so. Simulated
// devices for tests are in the separate module
// github.com/cB-Sigma/wiz-sim.
package wiz
