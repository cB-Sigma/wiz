// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

// Package wiz speaks the local UDP API of WiZ Wi-Fi lights and plugs: JSON
// requests to port 38899 of the device with resends (Request), the light
// state (Pilot) and system configuration, the module classes (colour,
// tunable white, dimmable white, plug) and built-in scenes, broadcast
// discovery (Discover), and push registration with the pushes devices
// send to port 38900 (Register, ServePush, PushedAddr). It needs no
// cloud account.
//
// The local API is not published by its vendor; this package follows the
// behaviour public clients describe (see PROVENANCE.md). Simulated
// devices for tests are in the separate module github.com/cB-Sigma/wiz-sim.
package wiz
