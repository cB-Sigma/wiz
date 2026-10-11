// Copyright 2026 Abhinav Gautam
// SPDX-License-Identifier: Apache-2.0

package wiz

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/cB-Sigma/lanbcast"
)

// Register asks the device at dev to push its state changes (syncPilot)
// and power-ups (firstBeat) to phoneIP, port 38900. It is sent from conn,
// the socket the pushes are read from, so the device's answer reaches that
// socket too. Devices forget a registration after a while; send it again
// every 20 seconds or so.
func Register(conn *net.UDPConn, dev *net.UDPAddr, phoneMAC, phoneIP string) error {
	msg, err := json.Marshal(map[string]any{"method": MethodRegistration, "params": RegistrationParams(phoneMAC, phoneIP, true)})
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(msg, dev)
	return err
}

// LocalIPFor returns the local IPv4 address the route to dev uses: the
// address to register when none is configured. "" when there is no route.
func LocalIPFor(dev *net.UDPAddr) string {
	c, err := net.DialUDP("udp4", nil, dev)
	if err != nil {
		return ""
	}
	defer c.Close()
	if la, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return la.IP.String()
	}
	return ""
}

// Push is one datagram a device pushed to the registered address.
type Push struct {
	// Method is MethodSyncPilot, MethodFirstBeat or MethodUpdateOta.
	// Handlers should ignore methods they do not know: later versions may
	// hand on more kinds.
	Method string
	// MAC is the device's MAC address, normalised; "" when the push did not
	// carry a valid one (updateOta progress need not carry one: tell the
	// device by From's IP).
	MAC string
	// Pilot is the state a syncPilot carries.
	Pilot Pilot
	// From is the datagram's source. A device pushes from a port of its
	// own, not from its API port: use PushedAddr to follow its address.
	From *net.UDPAddr

	// FwVersion is the firmware version a firstBeat carries; MatterVersion
	// and ProductID are carried by newer devices only.
	FwVersion     string
	MatterVersion string
	ProductID     *int
	// UpdateStatus is the number an updateOta progress push carries. Its
	// values are not publicly described.
	UpdateStatus *int
	// Params are the push's params as sent.
	Params map[string]any
}

// ServePush reads pushes from conn until conn is closed and calls handle
// for each, in order. A syncPilot is acknowledged to its sender as
// phoneMAC before handle runs, echoing its id and env as the vendor's
// own library does; datagrams that are not pushes (answers to requests
// sent from conn, for example) are skipped. It returns the read error
// that ended it.
func ServePush(conn *net.UDPConn, phoneMAC string, handle func(Push)) error {
	buf := make([]byte, 4096)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		var m Message
		if json.Unmarshal(buf[:n], &m) != nil {
			continue
		}
		switch m.Method {
		case MethodSyncPilot:
			if m.Params == nil {
				continue // an answer, not a push
			}
			b, _ := json.Marshal(m.Params)
			p, err := ParsePilot(b)
			if err != nil || p.MAC == "" {
				continue
			}
			env := m.Env
			if env == "" {
				env = "pro"
			}
			a := map[string]any{"method": MethodSyncPilot, "env": env, "result": map[string]any{"mac": phoneMAC}}
			if m.ID != nil {
				a["id"] = m.ID
			}
			ack, _ := json.Marshal(a)
			_, _ = conn.WriteToUDP(ack, from)
			handle(Push{Method: MethodSyncPilot, MAC: p.MAC, Pilot: p, From: from, Params: m.Params})
		case MethodFirstBeat:
			mac, fw, matter := "", "", ""
			var product *int
			if m.Params != nil {
				mac, _ = m.Params["mac"].(string)
				fw, _ = m.Params["fwVersion"].(string)
				matter, _ = m.Params["matterVersion"].(string)
				product = intOf(m.Params["productId"])
			}
			handle(Push{Method: MethodFirstBeat, MAC: NormaliseMAC(mac), From: from, FwVersion: fw, MatterVersion: matter, ProductID: product, Params: m.Params})
		case MethodUpdateOta:
			if m.Params == nil {
				continue // an answer to an updateOta sent from conn
			}
			mac, _ := m.Params["mac"].(string)
			handle(Push{Method: MethodUpdateOta, MAC: NormaliseMAC(mac), From: from, UpdateStatus: intOf(m.Params["updateStatus"]), Params: m.Params})
		}
	}
}

// AnswerAddr is a device's API address (host:port) taken from an answer
// to a request or to discovery: devices answer from their API port.
func AnswerAddr(from *net.UDPAddr) string {
	return net.JoinHostPort(from.IP.String(), strconv.Itoa(from.Port))
}

// PushedAddr is a device's API address after a push from it. A light
// pushes from a port of its own, not from its API port, so only the IP is
// taken from the push (which keeps a changed DHCP lease current); the port
// stays the one in prev, the address the device was known at, or 38899
// when prev has none. Taking the push's source port instead would send
// later requests to a port nothing listens on.
func PushedAddr(from *net.UDPAddr, prev string) string {
	port := strconv.Itoa(DevicePort)
	if _, p, err := net.SplitHostPort(prev); err == nil && p != "" {
		port = p
	}
	return net.JoinHostPort(from.IP.String(), port)
}

// Discover sends the discovery form of a registration to broadcast
// (host:port; "255.255.255.255:38899" also goes to the directed broadcast
// address of every LAN interface) three times, a second apart, and
// collects the answers until wait has passed or ctx ends. It returns each
// device's API address by normalised MAC.
func Discover(ctx context.Context, broadcast, phoneMAC string, wait time.Duration) (map[string]string, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	ba, err := net.ResolveUDPAddr("udp4", broadcast)
	if err != nil {
		return nil, err
	}
	msg, _ := json.Marshal(map[string]any{"method": MethodRegistration, "params": RegistrationParams(phoneMAC, "0.0.0.0", false)})
	deadline := time.Now().Add(wait)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetReadDeadline(time.Now()) })
	defer stop()
	found := map[string]string{}
	buf := make([]byte, 4096)
	for i := 0; ; i++ {
		// Three sends a second apart: a broadcast is lost more easily than
		// a unicast.
		if i < 3 {
			if err := lanbcast.WriteBroadcast(conn, msg, ba); err != nil && i == 0 {
				return nil, err
			}
		}
		next := time.Now().Add(time.Second)
		if next.After(deadline) {
			next = deadline
		}
		for {
			_ = conn.SetReadDeadline(next)
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				break
			}
			var m Message
			if json.Unmarshal(buf[:n], &m) != nil || m.Method != MethodRegistration {
				continue
			}
			var r struct {
				MAC string `json:"mac"`
			}
			_ = json.Unmarshal(m.Result, &r)
			if mac := NormaliseMAC(r.MAC); mac != "" {
				found[mac] = AnswerAddr(from)
			}
		}
		if !time.Now().Before(deadline) || ctx.Err() != nil {
			return found, nil
		}
	}
}
