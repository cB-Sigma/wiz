# wiz

A Go client for the local UDP API of WiZ lights and plugs; not affiliated with the owner of the WiZ brand.

[![Go Reference](https://pkg.go.dev/badge/github.com/cB-Sigma/wiz.svg)](https://pkg.go.dev/github.com/cB-Sigma/wiz) [![test](https://github.com/cB-Sigma/wiz/actions/workflows/test.yml/badge.svg)](https://github.com/cB-Sigma/wiz/actions/workflows/test.yml) [![Licence: Apache-2.0](https://img.shields.io/badge/licence-Apache--2.0-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/go-1.22-00ADD8.svg)](go.mod)

Requests with resends, light state and system configuration, module
classes and built-in scenes, broadcast discovery, and push registration
with the pushes the devices send. No cloud account, no cgo. The only
dependency outside the standard library is
[lanbcast](https://github.com/cB-Sigma/lanbcast) (Apache-2.0, standard
library only).

## Features

| Feature | Status |
|---|---|
| getPilot, setPilot (state, dimming, r/g/b, c/w, temp, sceneId, speed), getSystemConfig, getPower | yes; other methods can be sent with `Request` |
| Discovery (the discovery form of registration sent to a broadcast address; the limited broadcast also goes to each LAN's directed broadcast) | yes |
| Push registration, syncPilot and firstBeat pushes, acknowledgement | yes |
| Module classes | colour (RGB), tunable white, dimmable white, plug; fans and fan lights are reported as unknown |
| Scenes | the built-in scene numbers with their app names, per class |
| Wi-Fi setup of new devices, firmware update, WiZ Pro, the cloud API | no |

The vendor does not publish the local API; there is no standard to cite.
The behaviour follows public descriptions of it (see PROVENANCE.md).

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
| Fans | reported as unknown | no | no | no | yes |
| Dependencies outside the standard library | lanbcast | golang.org/x/exp | the D3iot module (gonum, a screenshot package) | go.uber.org/zap | Python |

Pick this module to find, control and follow WiZ devices from a Go
service: it is the only Go client found that discovers lights and
receives their pushes. Pick D3iot's driver when you want colours given in
CIE XYZ on the bulb it has measured. Use pywizlight in Python or Home
Assistant, and for fans.

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
for _, m := range []string{"ESP01_SHRGB1C_31", "ESP05_SHDW9_01", "ESP10_SOCKET_06", "ESP03_FANDIMS_01"} {
	c, ok := wiz.ClassOf(m)
	fmt.Println(m, c, ok)
}
// Output:
// ESP01_SHRGB1C_31 rgb true
// ESP05_SHDW9_01 dw true
// ESP10_SOCKET_06 socket true
// ESP03_FANDIMS_01 unknown false
```

Pushes: open a UDP socket on port 38900, `Register` each device from it
(again every 20 seconds or so), and run `ServePush` on it. Follow a
device's address with `PushedAddr`, never with a push's source port:
lights push from a port of their own.

## Documentation

[pkg.go.dev/github.com/cB-Sigma/wiz](https://pkg.go.dev/github.com/cB-Sigma/wiz)

| Package | What it does |
|---|---|
| `wiz` (module root) | `Request` (one method, resent with a growing gap until the timeout), `Pilot`/`ParsePilot`, `SystemConfig`/`ParseSystemConfig`, `ClassOf`, `Scenes`/`SceneID`/`SceneName`/`ScenesFor`/`HasScene`, `Discover`, `Register`, `ServePush`, `PushedAddr`, `LocalIPFor` |

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
bulbs and plugs have not been checked against real devices.

## Related repositories

- [wiz-sim](https://github.com/cB-Sigma/wiz-sim): simulated WiZ lights
  and a plug, as a Go package for tests and as a command.
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
