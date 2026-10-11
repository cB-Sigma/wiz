// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz_test

// Discovery, registration and pushes against simulated lights are tested
// in github.com/cB-Sigma/wiz-sim (sim/wiz_integration_test.go).

import (
	"net"
	"testing"

	"github.com/cB-Sigma/wiz"
)

func TestPushedAddrKeepsAPIPort(t *testing.T) {
	from := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 77), Port: 51234}
	for prev, want := range map[string]string{
		"192.168.1.41:38899": "192.168.1.77:38899", // a new lease: the IP follows, the port stays
		"127.0.0.1:40001":    "192.168.1.77:40001",
		"":                   "192.168.1.77:38899",
		"garbage":            "192.168.1.77:38899",
	} {
		if got := wiz.PushedAddr(from, prev); got != want {
			t.Errorf("PushedAddr(%q) = %s, want %s", prev, got, want)
		}
	}
	if got := wiz.AnswerAddr(from); got != "192.168.1.77:51234" {
		t.Errorf("AnswerAddr = %s", got)
	}
}
