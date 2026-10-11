package wiz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The WiZ local API: JSON objects in single UDP datagrams to port 38899 of
// the device, answered from that port. Pushes (syncPilot, firstBeat) go to
// port 38900 of the registered phone address.
const (
	DevicePort = 38899
	PushPort   = 38900
)

// Methods of the local API used here.
const (
	MethodGetPilot        = "getPilot"
	MethodSetPilot        = "setPilot"
	MethodGetSystemConfig = "getSystemConfig"
	MethodGetPower        = "getPower"
	MethodRegistration    = "registration"
	MethodSyncPilot       = "syncPilot"
	MethodFirstBeat       = "firstBeat"
)

// Message is one datagram of the local API.
type Message struct {
	Method string          `json:"method"`
	Env    string          `json:"env,omitempty"`
	ID     any             `json:"id,omitempty"`
	Params map[string]any  `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Pilot is the light state of getPilot, setPilot and syncPilot. Fields the
// device leaves out stay nil.
type Pilot struct {
	MAC     string   `json:"mac,omitempty"`
	State   *bool    `json:"state,omitempty"`
	Dimming *float64 `json:"dimming,omitempty"`
	R       *float64 `json:"r,omitempty"`
	G       *float64 `json:"g,omitempty"`
	B       *float64 `json:"b,omitempty"`
	C       *float64 `json:"c,omitempty"`
	W       *float64 `json:"w,omitempty"`
	Temp    *float64 `json:"temp,omitempty"`
	SceneID *float64 `json:"sceneId,omitempty"`
	Speed   *float64 `json:"speed,omitempty"`
	// Raw is every key as sent.
	Raw map[string]any `json:"-"`
}

// ParsePilot reads a pilot object.
func ParsePilot(b []byte) (Pilot, error) {
	var p Pilot
	if err := json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	_ = json.Unmarshal(b, &p.Raw)
	p.MAC = NormaliseMAC(p.MAC)
	return p, nil
}

// SystemConfig is the part of a getSystemConfig answer this package reads.
type SystemConfig struct {
	MAC        string `json:"mac"`
	ModuleName string `json:"moduleName"`
	FwVersion  string `json:"fwVersion"`
	TypeID     *int   `json:"typeId,omitempty"`
	HomeID     *int   `json:"homeId,omitempty"`
	RoomID     *int   `json:"roomId,omitempty"`
}

// ParseSystemConfig reads a getSystemConfig result; MAC is normalised
// ("" when it is not a MAC address).
func ParseSystemConfig(b []byte) (SystemConfig, error) {
	var c SystemConfig
	err := json.Unmarshal(b, &c)
	c.MAC = NormaliseMAC(c.MAC)
	return c, err
}

var macRE = regexp.MustCompile(`^[0-9a-f]{12}$`)

// NormaliseMAC lower-cases a MAC address and drops separators; "" when it
// is not 12 hexadecimal digits.
func NormaliseMAC(s string) string {
	s = strings.ToLower(strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(strings.TrimSpace(s)))
	if !macRE.MatchString(s) {
		return ""
	}
	return s
}

// Class is what a WiZ module can do, told from its module name.
type Class string

// The classes. The module name's second underscore-separated segment
// carries the class ("ESP01_SHRGB1C_31": RGB; "ESP56_SHTW3_01": tunable
// white; "ESP05_SHDW9_01": dimmable white; "ESP10_SOCKET_06": plug).
const (
	ClassRGB     Class = "rgb"    // colour, tunable white, dimming, scenes
	ClassTW      Class = "tw"     // tunable white, dimming, scenes
	ClassDW      Class = "dw"     // dimming, a few scenes
	ClassSocket  Class = "socket" // on and off, power where reported
	ClassUnknown Class = "unknown"
)

// ClassOf tells the class of a module name. ok is false for a module the
// package does not know (fans, a form it cannot read).
func ClassOf(module string) (c Class, ok bool) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(module)), "_")
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "ESP") {
		return ClassUnknown, false
	}
	seg := parts[1]
	switch {
	case strings.Contains(seg, "FAN"), strings.Contains(seg, "DDTW"):
		return ClassUnknown, false // ceiling fans with a light: not covered
	case strings.Contains(seg, "RGB"):
		return ClassRGB, true
	case strings.Contains(seg, "TW"):
		return ClassTW, true
	case strings.Contains(seg, "SOCKET"):
		return ClassSocket, true
	case strings.Contains(seg, "DW"):
		return ClassDW, true
	}
	return ClassUnknown, false
}

// Scene is a built-in scene (light mode) of WiZ lights.
type Scene struct {
	ID   int
	Name string
}

// Scenes are the built-in scene IDs of the WiZ local API with their names
// as the WiZ app shows them.
var Scenes = []Scene{
	{1, "Ocean"}, {2, "Romance"}, {3, "Sunset"}, {4, "Party"}, {5, "Fireplace"}, {6, "Cozy"},
	{7, "Forest"}, {8, "Pastel colors"}, {9, "Wake-up"}, {10, "Bedtime"}, {11, "Warm white"},
	{12, "Daylight"}, {13, "Cool white"}, {14, "Night light"}, {15, "Focus"}, {16, "Relax"},
	{17, "True colors"}, {18, "TV time"}, {19, "Plantgrowth"}, {20, "Spring"}, {21, "Summer"},
	{22, "Fall"}, {23, "Deep dive"}, {24, "Jungle"}, {25, "Mojito"}, {26, "Club"}, {27, "Christmas"},
	{28, "Halloween"}, {29, "Candlelight"}, {30, "Golden white"}, {31, "Pulse"}, {32, "Steampunk"},
	{33, "Diwali"}, {34, "White"}, {35, "Alarm"}, {36, "Snowy sky"}, {40, "Dim-to-warm"}, {1000, "Rhythm"},
}

// Scene IDs tunable-white and dimmable-white lights offer.
var (
	twScenes = []int{6, 9, 10, 11, 12, 13, 14, 15, 16, 18, 29, 30, 31, 32, 33, 35, 40}
	dwScenes = []int{9, 10, 14, 29, 31, 32, 34, 35}
)

// SceneName returns a scene's name; "" for 0 (no scene) and an unknown ID
// as "scene <id>".
func SceneName(id int) string {
	if id == 0 {
		return ""
	}
	for _, s := range Scenes {
		if s.ID == id {
			return s.Name
		}
	}
	return "scene " + strconv.Itoa(id)
}

func sceneKey(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.ToLower(s))
}

// SceneID finds a scene by name (case, spaces and hyphens ignored, so
// "wake up" is Wake-up) or by its number.
func SceneID(name string) (int, bool) {
	k := sceneKey(name)
	if k == "" {
		return 0, false
	}
	for _, s := range Scenes {
		if sceneKey(s.Name) == k {
			return s.ID, true
		}
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(k, "scene")); err == nil && n > 0 {
		return n, true
	}
	return 0, false
}

// ScenesFor lists the scene names a class of light offers.
func ScenesFor(c Class) []string {
	var ids []int
	switch c {
	case ClassRGB:
		for _, s := range Scenes {
			ids = append(ids, s.ID)
		}
	case ClassTW:
		ids = twScenes
	case ClassDW:
		ids = dwScenes
	default:
		return nil
	}
	sort.Ints(ids)
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = SceneName(id)
	}
	return out
}

// HasScene reports whether a class of light offers the scene.
func HasScene(c Class, id int) bool {
	switch c {
	case ClassRGB:
		return true
	case ClassTW:
		return containsInt(twScenes, id)
	case ClassDW:
		return containsInt(dwScenes, id)
	}
	return false
}

func containsInt(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// WithPort adds the default port to an address without one.
func WithPort(addr string, port int) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), strconv.Itoa(port))
}

// ErrDevice is wrapped by errors the device returned.
var ErrDevice = errors.New("the device refused the request")

// Request sends a message to a device and waits for the answer with the
// same method, resending with a growing gap (0.75 s, doubling, at most 3 s)
// until timeout: UDP loses datagrams and a bulb on busy Wi-Fi misses some.
func Request(ctx context.Context, addr, method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	msg := map[string]any{"method": method}
	if params != nil {
		msg["params"] = params
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	raddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetReadDeadline(time.Now()) })
	defer stop()
	gap := 750 * time.Millisecond
	buf := make([]byte, 4096)
	for {
		if _, err := conn.Write(b); err != nil {
			return nil, err
		}
		next := time.Now().Add(gap)
		if next.After(deadline) {
			next = deadline
		}
		for {
			_ = conn.SetReadDeadline(next)
			n, err := conn.Read(buf)
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					break
				}
				// An ICMP port unreachable surfaces as a read error; wait
				// out the gap before sending again.
				time.Sleep(time.Until(next))
				break
			}
			var m Message
			if json.Unmarshal(buf[:n], &m) != nil || m.Method != method {
				continue
			}
			if m.Error != nil {
				return nil, fmt.Errorf("%w: %s: %d %s", ErrDevice, method, m.Error.Code, m.Error.Message)
			}
			return m.Result, nil
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("%s: no answer from %s within %s", method, addr, timeout)
		}
		gap *= 2
		if gap > 3*time.Second {
			gap = 3 * time.Second
		}
	}
}

// RegistrationParams are the params of a registration: phoneMac (12
// hexadecimal digits), phoneIp (where pushes go) and register (true asks
// for pushes; false is the discovery form).
func RegistrationParams(phoneMAC, phoneIP string, register bool) map[string]any {
	return map[string]any{"phoneMac": phoneMAC, "register": register, "phoneIp": phoneIP, "id": "1"}
}
