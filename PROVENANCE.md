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

## Sources

| Area (files) | Followed | Read for facts only (no code taken) |
|---|---|---|
| Requests, pilot, system config, classes, scenes (proto.go) | none published | pywizlight (github.com/sbidy/pywizlight, MIT): the method and field names, the ports, the module-name classes (RGB, TW, DW, SOCKET; fans not covered) and the scene table |
| Registration, pushes, discovery (push.go) | none published | as above: registration parameters (phoneMac, phoneIp, register), syncPilot and firstBeat pushes to port 38900 and their acknowledgement, the discovery form of registration sent to the broadcast address |
| Broadcast helper (the lanbcast module) | none | none; our own code |

## Passages to review

No passage is known to be copied. These are the places closest to a
public source, because they are tables of facts that every client holds
in the same form:

- `Scenes` in proto.go: the scene numbers with the names the WiZ app
  shows. The names and numbers are facts; the order follows the numbers.
- `twScenes` and `dwScenes` in proto.go: which scenes tunable-white and
  dimmable-white lights offer.
- `ClassOf` in proto.go: the module-name segments that tell the class.

A line-by-line comparison with pywizlight's source has not been done.

## Test data

All MAC addresses, module names, firmware versions and addresses in the
tests and examples are made-up demo values. The module names are real
model-name forms, chosen to exercise each class.
