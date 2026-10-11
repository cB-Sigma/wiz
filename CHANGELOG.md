# Changelog

All notable changes to this module. Versions follow semantic versioning;
before v1 the API may change in minor versions.

## v0.3.0 (2026-10-11)

### Added

- Per-head control of dual-head lamps, with pywizlight 0.6.6's
  numbering: `HeadParams`, `GetHeadPilot`, `ErrNoHeadState`,
  `HeadFirst`/`HeadSecond`, `Pilot.Devices` and `Pilot.Head`.
- pywizlight's colour mapping as an option: `HSToRGBWPywizlight`,
  `RGBToRGBWPywizlight`, and `RGBCW.RGBWParams` (r, g, b and w, without
  c).
- `Pulse`, `PulseParams`, `MethodPulse`, `MaxPulseDelta`.
- `Pilot.SchdPsetID`, `Pilot.EffectiveSceneID` and `SceneRhythm`.
- The interop module: this client against the fake bulbs of pywizlight
  0.6.6's test suite, in Docker (SIGMA_WIZ_INTEROP=1).
- README: per-head numbering, "Not yet" (items needing hardware or a
  vendor account), new comparison rows.

### Fixed

- `ParseSystemConfig` no longer returns an error when typeId, homeId or
  roomId is not a number (pywizlight's recorded answers carry redacted
  strings); those fields are then nil. Found by the interop test.

## v0.2.0 (2026-10-11)

### Added

- `HSToRGBCW`, `RGBToRGBCW` and `RGBCW`: colour from hue and saturation
  or from RGB to the five setPilot channels (our own mapping).
- Fan modules: classes `ClassFanDim` (FANDIM, a fan with a dimmable
  light) and `ClassFanTW` (DDTW, a fan with a tunable-white light);
  `FanParams` with `FanModeNormal`/`FanModeBreeze`, speed checked against
  `FanSpeedMax` (the device's fanSpeed, else `DefaultFanSpeedMax`, 6).
- `FeaturesOf`: brightness, colour, colour temperature, scenes, fan,
  breeze, reverse and dual head per module.
- `ClassOfTypeID` and `ClassOfConfig` for old firmware without a module
  name (typeId 0: dimmable white).
- `Pilot` fields `FanState`, `FanMode`, `FanSpeed`, `FanRevrs`, `Ratio`
  and `PC` (power in milliwatts); `RatioParams` for dual-head lamps.
- Method constants for setState, getModelConfig, getUserConfig,
  getWifiConfig, setUserConfig, setSystemConfig, setModelConfig, reboot,
  reset and updateOta; `CodeMethodNotFound` and `CodeInvalidParams`.
- `ModelConfig`/`ParseModelConfig`/`GetModelConfig`,
  `UserConfig`/`ParseUserConfig`/`GetUserConfig`, `KelvinRange` (with
  `DefaultKelvinMin`/`DefaultKelvinMax`), `UserConfigParams` and
  `SetUserConfig`, `GetWifiConfig` (result as sent), and the raw, documented
  as dangerous, `SetSystemConfig` and `SetModelConfig`.
- `SystemConfig` fields filled by `ParseSystemConfig`: `HomeLock`,
  `PairingLock`, `Lock`, `GroupID`, `Ping`, `FwOtaStatus`, `Region`,
  `EWFHex`, `WhiteToColorRatio` and `WhiteChannels` (from drvConf).
- `GetPower` (milliwatts) and `ErrNoReading`.
- `Reboot`, `Reset` and `UpdateFirmware`, each sent once without resends.
- `DeviceError`, the type of the error a refusing device answers with; it
  wraps `ErrDevice` and has the same text as before.
- `Push` fields `FwVersion`, `MatterVersion`, `ProductID` (firstBeat),
  `UpdateStatus` (updateOta progress) and `Params`.
- CHANGELOG.md; README sections on what is not implemented and why.

### Changed

- `ClassOf` checks the module-name segment for RGB, DDTW, TW, SOCKET,
  FANDIM, then DW. Fan modules, which were unknown (`ok` false), now have
  their classes; a segment holding RGB and DDTW is now colour.
- `ServePush` also hands on updateOta progress pushes (`Push.Method` is
  `MethodUpdateOta`); handlers should ignore methods they do not know.
- The syncPilot acknowledgement echoes the push's `id`, and its `env`
  when it has one (it was always "pro").

## v0.1.1

Inferred from the repository history; the release date is not recorded
here. The simulator moved to its own module, github.com/cB-Sigma/wiz-sim
(2026-10-11); the library's own tests use a small fake light, and the
discovery, push and request tests against the simulator moved with it.

## v0.1.0

Inferred from the repository history; the release date is not recorded
here. First release as a module of its own, split from SigmaM2M
(2026-10-06): `Request` with resends, `Pilot`, `SystemConfig`, `ClassOf`,
the scene table, `Discover`, `Register`, `ServePush`, `PushedAddr` and
`LocalIPFor`.
