# Provenance

## How the code was written

Written in Go for SigmaM2M, a oneM2M platform. No code was ported or
copied from another implementation, and no source file of another
project was translated.

The vendor does not publish the WiZ local UDP API, so there is no
standard to follow. The facts this module uses (port 38899 for requests
and 38900 for pushes; the JSON message shape with `method`, `params`,
`result` and `error`; the method names getPilot, setPilot,
getSystemConfig, getPower, registration, syncPilot and firstBeat; the
pilot fields; the registration parameters; the module-name segments that
tell a light's class; the built-in scene numbers and their names in the
WiZ app) were taken from public descriptions of the API, chiefly the
pywizlight project and its documentation. They are interface facts that
every client must use. The Go types, the resend schedule, the push and
discovery helpers, the error handling and the simulator are our own
design.

proto.go came from SigmaM2M's first version (2026-09-27). The
push-address handling (`PushedAddr`: a push's source IP is taken, its
source port is not, because lights push from a port of their own) was
corrected on 2026-10-05, after a client was found to take the push's
source port as the light's. push.go and the tests marked "Copyright 2026
Abhinav Gautam" were written for this module from SigmaM2M's own code.
The simulator moved to its own module, wiz-sim, on 2026-10-11.

### v0.2.0 (2026-10-11)

v0.2.0 adds fans, dual-head ratio, setState, the configuration and
maintenance methods, power in the pilot, firmware-update pushes and the
richer firstBeat. Its facts were read, for facts only, from the four
public sources below, at the commits named, on 2026-10-11. Each fact used
was checked against a saved copy of the source; the wording, types and
structure here are our own. The fourth source,
wiz-local-control, is the vendor's own library (MIT, Copyright (c) 2018
WiZLighting); it was read for facts the same way and no code was taken
from it. A line-by-line comparison of our code with these sources was not
needed, at the user's direction: nothing was translated from them.

## Sources

| Source | Commit read | Licence |
|---|---|---|
| [PY] pywizlight, github.com/sbidy/pywizlight | master c6ce9d97677aab1da3a693973fb3487383f5d235 (latest tag v0.6.6) | MIT, Copyright (c) 2020 Stephan Traub |
| [HA] Home Assistant, github.com/home-assistant/core, homeassistant/components/wiz | dev 8792c68d | Apache-2.0 |
| [OH] openHAB add-ons, github.com/openhab/openhab-addons, bundles/org.openhab.binding.wiz | main a5a74fd0 | EPL-2.0 |
| [WZ] wiz-local-control, gitlab.com/wizlighting/wiz-local-control (the vendor's own library) | master b7bfe09f (2025-09-03) | MIT, Copyright (c) 2018 WiZLighting |

| Area (files) | Followed | Read for facts only (no code taken) |
|---|---|---|
| Requests, pilot, system config, classes, scenes (proto.go) | none published | PY: the method and field names, the ports, the module-name classes and the scene table |
| Registration, pushes, discovery (push.go) | none published | PY: registration parameters (phoneMac, phoneIp, register), syncPilot and firstBeat pushes to port 38900 and their acknowledgement, the discovery form of registration sent to the broadcast address. WZ: the acknowledgement echoes the push's id and env; firstBeat's fwVersion, matterVersion and productId; updateOta progress messages with updateStatus |
| Fan classes and keys, features (proto.go, features.go) | none published | PY: the class order RGB, DDTW, TW, SOCKET, FANDIM, then DW; the feature table per class; effects for dimmable white only with SH or DH; DH for dual head; fanState 0/1, fanMode 1/2, fanSpeed 1 to the reported count (6 by default), fanRevrs 0/1; ratio 0 to 100; setState not switching the light on. HA: breeze sends fanMode 2 alone, a speed goes with fanMode 1, any pilot with ratio is ratio-capable. OH: the example module ESP03_FANDIMS_31, speed 0 meaning off |
| Configuration (config.go) | none published | PY: getModelConfig before getUserConfig, the kelvin range from extRange else cctRange as min to max, fanSpeed from either, drvConf as [ratio, white channels] on firmware before 1.22, typeId 0 as dimmable white, the field names of both answers (with sample shapes: flags as numbers, renderFactor as a list). WZ: setUserConfig params (whiteRange, extRange, pwmRange, dftDim, startupMode, userConfigTs defaulting to 0); setSystemConfig params (env, moduleName, ewf, pwmConf, drvConf, fs, systemConfigTs); setModelConfig params and which are required; getSystemConfig's fwOtaStatus. OH: getWifiConfig's result noted as an encrypted string; the 2200 to 6500 K default |
| Maintenance, power (maintenance.go) | none published | PY: reboot and reset with empty params; getPower's result.power in milliwatts, pc in the pilot, "method not found" without metering. WZ: updateOta with fw ("default" when none) and force 0/1 |
| Colour conversion (color.go) | the usual hue-to-RGB sextant formula and the common practice of moving an RGB colour's grey part to white LEDs | none; our own mapping, written without reading pywizlight's rgbcw.py |
| Broadcast helper (the lanbcast module) | none | none; our own code |

Not implemented, because no public source describes enough to implement
it honestly: Wi-Fi setup (setWifiConfig is only a name in PY's README
and OH), pulse and other methods named without parameters, per-head
control of dual-head lamps (PY's development branch numbers the heads
inconsistently), WiZ Pro (a GraphQL cloud API at
docs.pro.wizconnected.com that needs a client ID from WiZ) and the
consumer WiZ cloud (no public API found).

## Passages to review

No passage is known to be copied. These are the places closest to a
public source, because they are tables of facts that every client holds
in the same form:

- `Scenes` in proto.go: the scene numbers with the names the WiZ app
  shows. The names and numbers are facts; the order follows the numbers.
- `twScenes` and `dwScenes` in proto.go: which scenes tunable-white and
  dimmable-white lights offer.
- `ClassOf` in proto.go and `FeaturesOf` in features.go: the
  module-name segments that tell the class, their order, and the
  features per class.

A line-by-line comparison with pywizlight's source has not been done.

## Test data

All MAC addresses, module names, firmware versions and addresses in the
tests and examples are made-up demo values. The module names are real
model-name forms, chosen to exercise each class.
