# wiz

A Go client for the local UDP API of WiZ lights and plugs; not affiliated with the owner of the WiZ brand.

[![Go Reference](https://pkg.go.dev/badge/github.com/cB-Sigma/wiz.svg)](https://pkg.go.dev/github.com/cB-Sigma/wiz) [![test](https://github.com/cB-Sigma/wiz/actions/workflows/test.yml/badge.svg)](https://github.com/cB-Sigma/wiz/actions/workflows/test.yml) [![Licence: Apache-2.0](https://img.shields.io/badge/licence-Apache--2.0-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/go-1.22-00ADD8.svg)](go.mod)

Requests with resends, light, fan and plug state, configuration,
maintenance (reboot, factory reset, firmware update), module classes and
their features, built-in scenes, broadcast discovery, and push
registration with the pushes the devices send. No cloud account, no cgo.
The only dependency outside the standard library is
[lanbcast](https://github.com/cB-Sigma/lanbcast) (Apache-2.0, standard
library only).

## Features

| Feature | Status |
|---|---|
| getPilot, setPilot and setState (state, dimming, r/g/b, c/w, temp, sceneId, speed, ratio, the fan keys); setState sets the same keys without switching the light on | yes; any other method can be sent with `Request` |
| Module classes | colour (RGB), tunable white, dimmable white, plug, ceiling fan with a dimmable light (FANDIM), ceiling fan with a tunable-white light (DDTW); `ClassOfTypeID` for old firmware without a module name |
| Colour from hue and saturation or RGB | `HSToRGBCW`, `RGBToRGBCW`: our own mapping to r, g, b and the two white LEDs (the grey part goes to the white LEDs), not a measured colour model |
| Features per module (`FeaturesOf`) | brightness, colour, colour temperature, scenes, fan, breeze mode, reverse, dual head |
| Fans | yes: fanState, fanMode (normal, breeze), fanSpeed checked against the device's number of speeds, fanRevrs (`FanParams`, `FanSpeedMax`) |
| Dual-head lamps | the up/down split (`ratio`, `RatioParams`); per-head control (the `devices` key seen in pywizlight's development branch) is not implemented, as its numbering is not consistently described |
| Power | getPower in milliwatts (`GetPower`), `pc` in the pilot on newer firmware; devices without metering answer "method not found"; there is no energy counter in the local API |
| Configuration | getSystemConfig (with locks, group, region, drvConf, fwOtaStatus), getModelConfig, getUserConfig, the kelvin range (`KelvinRange`), setUserConfig with the params the vendor's library describes, getWifiConfig as sent |
| Factory settings | setSystemConfig and setModelConfig with params as given: they can misconfigure the driver, so there are no helpers that pick values |
| Maintenance | `Reboot`, `Reset` (factory reset) and `UpdateFirmware`, each sent once; firmware-update progress arrives as updateOta pushes |
| Discovery (the discovery form of registration sent to a broadcast address; the limited broadcast also goes to each LAN's directed broadcast) | yes |
| Push registration; syncPilot, firstBeat (firmware, Matter version, product ID) and updateOta pushes; acknowledgement echoing the push's id | yes |
| Scenes | the built-in scene numbers with their app names, per class (fan lights: no published list) |
| Wi-Fi setup of new devices | no: see below |
| WiZ Pro, the WiZ cloud | no: see below |

The vendor does not publish the local API; there is no standard to cite.
The behaviour follows public descriptions of it (see PROVENANCE.md).

### What is not implemented, and why

- **Wi-Fi setup of a new device** (in its access-point mode). The method
  that would do it, setWifiConfig, appears in public descriptions only as
  a name; its parameters and the setup sequence are not publicly
  documented. Implementing it would need capturing the WiZ app's traffic
  during setup. Set devices up with the WiZ app.
- **Firmware images of one's own.** `UpdateFirmware` asks the device to
  fetch a version; the device fetches the image itself (presumably from
  the vendor's servers), so no image can be supplied. The numbers in the
  progress pushes (`updateStatus`) and in `fwOtaStatus` are reported as
  sent: their meanings are not publicly described.
- **WiZ Pro.** Its API is a GraphQL cloud API
  ([docs.pro.wizconnected.com](https://docs.pro.wizconnected.com)) with
  OAuth (PKCE) sign-in that needs a client ID issued by WiZ; there is no
  public sign-up for one, so it cannot be implemented or tested without a
  vendor account.
- **The consumer WiZ cloud.** No public API for it was found.
- **pulse and other named-only methods** (getDevInfo and others): public
  descriptions name them without parameters that could be checked.
  `Request` sends them as given.
- **Meanings not described anywhere public**: the user config's `po`, the
  values of `startupMode`, and the content of getWifiConfig (openHAB notes
  an encrypted string, unverified). They are passed through as sent.

Ranges, as public descriptions give them (they disagree in places, so
this package checks only where stated): dimming 1 to 100 (the vendor's
library; older descriptions say 10), temp 1000 to 10000 K (pywizlight) or
1500 to 9000 K (the vendor's library), r, g, b, c, w 0 to 255, sceneId 1
to 150 plus 1000 (Rhythm), speed 10 to 200 (pywizlight, Home Assistant)
or 20 to 200 (the vendor's library), ratio 0 to 100, fanSpeed 1 to the
number of speeds the device reports (6 when it reports none).

## Choosing a WiZ client

Compared from each library's source at the versions named, in October
2026; pywizlight is in Python and listed for reference. Check the
projects themselves before relying on a row.

| | this module | achetronic/wizgo v0.1.0 | Dadido3/D3iot wiz driver (2022 commit) | ctrox/gowiz (2021 commit) | pywizlight 0.6.6 (Python) |
|---|---|---|---|---|---|
| Licence | Apache-2.0 | Apache-2.0 | MIT | MIT | MIT |
| Last release or commit | v0.x, 2026 | tag 2023-09-19, last commit 2026-06-30 | 2022-11-23 | 2021-11-24 | 2026-08-05 |
| Timeout on a request | yes, `context.Context` | no (the read blocks) | yes | yes | yes |
| Resends a lost request | yes, with a growing gap | no | yes, 10 retries by default | no | yes, up to 6 sends |
| Discovery by broadcast | yes, limited and every LAN's directed broadcast | no | no | no | yes |
| Push registration | yes | sends the request | no | no | yes |
| Receives syncPilot pushes | yes (`ServePush`) | no | no | no | yes |
| Class from the module name, scenes per class | yes | yes | class for three listed products, colour set in CIE XYZ (one bulb measured); a scene list without classes | no | yes |
| Fans | yes (v0.2.0) | no | no | no | yes |
| Dependencies outside the standard library | lanbcast | golang.org/x/exp | the D3iot module (gonum, a screenshot package) | go.uber.org/zap | Python |

Pick this module to find, control and follow WiZ devices, fans included,
from a Go service: it is the only Go client found that discovers lights
and receives their pushes. Pick D3iot's driver when you want colours
given in CIE XYZ on the bulb it has measured. Use pywizlight in Python or
Home Assistant, when you want its own conversion of hue and saturation to
the bulbs' channels (this module has a simpler one of its own,
`HSToRGBCW` and `RGBToRGBCW`, not a measured colour model), or for
per-head control of dual-head lamps (in its development branch).

Beyond the table, this module (v0.2.0) also has the configuration,
reboot, factory reset and firmware-update calls; of the others, only
pywizlight was checked for them (it has getModelConfig, getUserConfig,
reboot and reset, and no firmware update).

## Install

```
go get github.com/cB-Sigma/wiz@latest
```

Requires Go 1.22 or later.

## Quick start

Find the lights on the LAN, read each one's module and switch it to a
scene (from `Example` in example_test.go; it needs lights on the LAN, so
it is compiled but not run):

```go
ctx := context.Background()

// The MAC is the one this client reports as its own.
found, err := wiz.Discover(ctx, "255.255.255.255:38899", "02aabbccddee", time.Second)
if err != nil {
	return err
}
for mac, addr := range found {
	raw, err := wiz.Request(ctx, addr, wiz.MethodGetSystemConfig, nil, 3*time.Second)
	if err != nil {
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
```

Telling a device's class from its module name (`ExampleClassOf`, run by
`go test`):

```go
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
```

Switching a ceiling fan on at speed 3, checked against the number of
speeds the fan reports; its light is not touched (`ExampleFanParams`; it
needs a fan on the LAN, so it is compiled but not run):

```go
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
```

`ExampleFeaturesOf` and `ExampleKelvinRange` (run by `go test`) show the
features of a module and the colour-temperature range read from the
configuration.

Pushes: open a UDP socket on port 38900, `Register` each device from it
(again every 20 seconds or so), and run `ServePush` on it. Follow a
device's address with `PushedAddr`, never with a push's source port:
lights push from a port of their own. `ServePush` hands on syncPilot,
firstBeat and updateOta (firmware-update progress) pushes; ignore methods
your handler does not know.

## Documentation

[pkg.go.dev/github.com/cB-Sigma/wiz](https://pkg.go.dev/github.com/cB-Sigma/wiz)

| Package | What it does |
|---|---|
| `wiz` (module root) | `Request` (one method, resent with a growing gap until the timeout; refusals as `*DeviceError`), `Pilot`/`ParsePilot`, `SystemConfig`/`ParseSystemConfig`, `ClassOf`/`ClassOfTypeID`/`ClassOfConfig`, `FeaturesOf`, `FanParams`/`FanSpeedMax`, `RatioParams`, `HSToRGBCW`/`RGBToRGBCW`, `ModelConfig`/`UserConfig` with `GetModelConfig`/`GetUserConfig`/`KelvinRange`, `SetUserConfig`, `SetSystemConfig`/`SetModelConfig` (raw, dangerous), `GetWifiConfig`, `GetPower`, `Reboot`, `Reset`, `UpdateFirmware`, `Scenes`/`SceneID`/`SceneName`/`ScenesFor`/`HasScene`, `Discover`, `Register`, `ServePush`, `PushedAddr`, `LocalIPFor` |

Changes per version are in [CHANGELOG.md](CHANGELOG.md).

## Testing

```
go test -race ./...
```

The library's tests use a small fake light on the loopback interface.
Discovery, registration and pushes against the full simulator are tested
in [wiz-sim](https://github.com/cB-Sigma/wiz-sim).

Tests carry `// Covers: SIG-...` comments. These are requirement IDs of
the compliance report of SigmaM2M, the oneM2M platform this module was
written for; they have no meaning outside it.

### Interop tested against

The automated tests run against a fake light and the simulator in
wiz-sim, both written from the same descriptions as the client; that is
not independent.

Checked by hand against a real WiZ colour bulb on a home LAN (October
2026, through SigmaM2M): colour, dimming at 20% and 100%, off and on, and
scene selection. Discovery, pushes, tunable-white and dimmable-white
bulbs and plugs have not been checked against real devices, and nothing
added in v0.2.0 has (fans, dual-head ratio, setState, the configuration
calls, reboot, reset, firmware update, power in the pilot): those follow
the public descriptions and are tested only against the simulator.

## Related repositories

- [wiz-sim](https://github.com/cB-Sigma/wiz-sim): simulated WiZ lights,
  a plug and ceiling fans, as a Go package for tests and as a command.
- [lanbcast](https://github.com/cB-Sigma/lanbcast): the LAN broadcast
  helper used by `Discover`.

## Status

v0.x; the API may change before v1.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## Licence

Apache-2.0 (LICENSE). Copyright Abhinav Gautam. NOTICE lists the
projects whose public descriptions supplied the protocol facts. WiZ is a
trademark of its owner; the name is used only to say which devices this
module talks to.
