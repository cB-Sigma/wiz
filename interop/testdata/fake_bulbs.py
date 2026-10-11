# Copyright 2026 Abhinav Gautam
# SPDX-License-Identifier: Apache-2.0
#
# Starts the fake bulbs of pywizlight's own test suite
# (pywizlight.tests.fake_bulb, installed with the pinned pywizlight
# package) inside the container, one per module and firmware named on the
# command line, and relays each to a port on all interfaces so that the
# Go client outside the container can reach it (the fake bulbs listen on
# the container's 127.0.0.1 only). The relay forwards datagrams unchanged
# in both directions. Our own script; it contains no pywizlight code.
#
# Output, one JSON line each: {"bulb": i, "module", "fw", "port",
# "bulbtype": pywizlight's own reading of the bulb} for every bulb, then
# {"ready": true}. The script runs until stdin closes or reads "quit".

import argparse
import asyncio
import json
import sys

from pywizlight import wizlight
from pywizlight._version import __version__ as pywizlight_version
from pywizlight.tests.fake_bulb import startup_bulb


class Upstream(asyncio.DatagramProtocol):
    def __init__(self, relay):
        self.relay = relay

    def datagram_received(self, data, addr):
        if self.relay.client is not None:
            self.relay.transport.sendto(data, self.relay.client)


class Relay(asyncio.DatagramProtocol):
    def __init__(self):
        self.transport = None
        self.client = None
        self.upstream = None

    def connection_made(self, transport):
        self.transport = transport

    def datagram_received(self, data, addr):
        self.client = addr
        if self.upstream is not None:
            self.upstream.sendto(data)


async def bulbtype(port):
    light = wizlight("127.0.0.1", port=port)
    try:
        bt = await asyncio.wait_for(light.get_bulbtype(), 20)
        return bt.as_dict()
    except Exception as ex:  # noqa: BLE001
        return {"error": f"{type(ex).__name__}: {ex}"}
    finally:
        await light.async_close()


async def run(a):
    loop = asyncio.get_event_loop()
    print(json.dumps({"pywizlight": pywizlight_version}), flush=True)
    for i, spec in enumerate(a.bulbs):
        module, fw = spec.split("@")
        _, port = await startup_bulb(module_name=module, firmware_version=fw)
        relay = Relay()
        await loop.create_datagram_endpoint(lambda r=relay: r, local_addr=("0.0.0.0", a.base + i))
        up, _ = await loop.create_datagram_endpoint(lambda r=relay: Upstream(r), remote_addr=("127.0.0.1", port))
        relay.upstream = up
        bt = {} if module == "BROKEN_JSON" else await bulbtype(port)
        print(json.dumps({"bulb": i, "module": module, "fw": fw, "port": a.base + i, "bulbtype": bt}), flush=True)
    print(json.dumps({"ready": True}), flush=True)
    while True:
        line = await loop.run_in_executor(None, sys.stdin.readline)
        if not line or line.strip() == "quit":
            return


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--base", type=int, default=40000)
    p.add_argument("bulbs", nargs="+", help="module@firmware")
    asyncio.run(run(p.parse_args()))


if __name__ == "__main__":
    main()
