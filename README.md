# raknet

RakNet for Minecraft: Bedrock Edition, with full low-level access for servers.

**Status:** in development. The API will change as low-level hooks are added.

Part of [Helix](https://github.com/helix-bedrock), an open-source Minecraft: Bedrock
Edition server engine in Go. Helix maintains this library.

## Origin

This library started as a copy of [go-raknet](https://github.com/Sandertv/go-raknet)
by **Sander ten Veldhuis** (MIT), taken from a fork that added congestion control,
transport metrics and a larger test suite. Thank you to Sander for the original work.
Helix maintains and changes it from here on.

## Install

```bash
go get github.com/helix-bedrock/raknet
```

## Example

```go
listener, err := raknet.Listen("0.0.0.0:19132")
if err != nil {
    return err
}
defer listener.Close()

for {
    conn, err := listener.Accept()
    if err != nil {
        return err
    }
    go handle(conn)
}
```

## Hooks

Hooks give full low-level access to every stage of a connection. Add or remove
them at any time; a hook that is not registered costs nothing (about 2 ns and no
allocations per check). Hooks can observe, change (`e.Data = ...`) or drop
(`return raknet.Drop`) what passes through them, and a panicking hook is logged
instead of crashing the server.

```go
// Every connection on the listener.
remove := listener.OnPacketIn(func(e *raknet.PacketEvent) raknet.Verdict {
    log.Printf("%v sent %d bytes", e.Conn.RemoteAddr(), len(e.Data))
    return raknet.Allow
})
defer remove()

// Reject connection attempts before a connection exists.
listener.OnConnectionAttempt(func(e *raknet.AttemptEvent) raknet.Verdict {
    if banned(e.Addr) {
        return raknet.Drop
    }
    return raknet.Allow
})

// Just one connection.
conn.OnPacketOut(func(e *raknet.PacketEvent) raknet.Verdict { return raknet.Allow })
```

| Hook | Listener | Conn | Sees |
|---|---|---|---|
| `OnRawIn` / `OnRawOut` | ✓ | ✓ | Every UDP datagram, with address |
| `OnOffline` | ✓ | | Pings and handshake messages before a connection exists |
| `OnConnectionAttempt` | ✓ | | Peers that passed the handshake checks: GUID, MTU |
| `OnConnect` / `OnDisconnect` | ✓ | `OnDisconnect` | Connection lifecycle |
| `OnFrameIn` | ✓ | ✓ | Each frame: reliability, indexes, split info, payload |
| `OnPacketIn` / `OnPacketOut` | ✓ | ✓ | Complete packets after reassembly / before splitting |

## Contributing

See the [contributing guide](https://github.com/helix-bedrock/.github/blob/main/CONTRIBUTING.md).

## License

MIT. See [`LICENSE`](LICENSE). The original copyright notice from go-raknet is kept,
as the MIT license requires. Sources are listed in [`CREDITS.md`](CREDITS.md).
