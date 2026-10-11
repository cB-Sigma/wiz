// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrNoReading is returned when an answer carries no value where one was
// expected, such as a getPower answer without power.
var ErrNoReading = errors.New("the answer carried no reading")

// GetPower asks a metering device (a plug) for its power draw and returns
// it in milliwatts. A device without metering answers CodeMethodNotFound
// (a *DeviceError). Newer firmware also reports the draw in the pilot
// (Pilot.PC). The local API has no energy counter.
func GetPower(ctx context.Context, addr string, timeout time.Duration) (milliwatts float64, err error) {
	raw, err := Request(ctx, addr, MethodGetPower, nil, timeout)
	if err != nil {
		return 0, err
	}
	var r struct {
		Power *float64 `json:"power"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &r) != nil || r.Power == nil {
		return 0, fmt.Errorf("getPower: %w", ErrNoReading)
	}
	return *r.Power, nil
}

// Reboot restarts a device. The request is sent once, not resent: a resend
// could reach the device after it has restarted. The device is silent
// while it restarts and announces itself with a firstBeat push afterwards.
func Reboot(ctx context.Context, addr string, timeout time.Duration) error {
	_, err := request(ctx, addr, MethodReboot, map[string]any{}, timeout, false)
	return err
}

// Reset sets a device back to its factory settings. Sent once, as Reboot.
// What the device keeps (its Wi-Fi network, its pairing) is not publicly
// described; expect to have to set it up again with the WiZ app.
func Reset(ctx context.Context, addr string, timeout time.Duration) error {
	_, err := request(ctx, addr, MethodReset, map[string]any{}, timeout, false)
	return err
}

// UpdateFirmware asks a device to update its firmware (updateOta) to the
// version fw, or to the version the vendor offers when fw is "" (sent as
// "default"); force is sent as 1 or 0, as the vendor's own library does.
// The device fetches the image itself, presumably from the vendor's
// servers: an image of one's own cannot be given. The request is sent
// once, and the result is returned as sent. Progress arrives as updateOta
// pushes at the address registered for pushes (Push.UpdateStatus, whose
// values are not publicly described); getSystemConfig reports
// fwOtaStatus, and the new fwVersion once the device has restarted.
func UpdateFirmware(ctx context.Context, addr, fw string, force bool, timeout time.Duration) (json.RawMessage, error) {
	if fw == "" {
		fw = "default"
	}
	return request(ctx, addr, MethodUpdateOta, map[string]any{"fw": fw, "force": b01(force)}, timeout, false)
}
