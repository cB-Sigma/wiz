// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

// Package interop holds tests only: the wiz client against the fake bulbs
// that pywizlight's own test suite ships (pywizlight/tests/fake_bulb.py,
// MIT), a device written independently of this module, with answers
// recorded per module and firmware by pywizlight's authors (colour,
// tunable white, dimmable white, plugs
// with and without metering, a fan, a dual-head lamp, firmware 1.8.0
// without a module name, a bulb that answers broken JSON). It checks the
// system, model and user configuration, the class, the kelvin range and
// the fan speeds against pywizlight's own reading of the same bulb, the
// pilot, setPilot, power, registration and the discovery answer.
//
// The fake bulbs run in a pinned Python image, which installs the pinned
// pywizlight package from PyPI (the fake bulbs come with it) and runs
// testdata/fake_bulbs.py, our own script that starts them and relays each
// to a published port (they listen on the container's loopback only). No
// pywizlight code is kept in this repository. The tests run only when
// SIGMA_WIZ_INTEROP=1 and docker is on PATH, and need network access for
// the package:
//
//	cd interop && SIGMA_WIZ_INTEROP=1 GOWORK=off go test -v -timeout 10m ./...
//
// The fake bulbs answer only getPilot, setPilot, getSystemConfig,
// getModelConfig, getUserConfig, getPower and registration, and send no
// pushes; the rest of the client is checked against wiz-sim and, through
// wiz-sim's interop module, pywizlight's client.
package interop
