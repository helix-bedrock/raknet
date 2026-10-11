# raknet

RakNet for Minecraft: Bedrock Edition, with full low-level access for servers.

**Status:** in development. The API will change as low-level hooks are added.

Part of [Helix](https://github.com/helix-bedrock), an open-source Minecraft: Bedrock
Edition server engine in Go. Helix maintains this library.

## Origin

This library started as a copy of [go-raknet](https://github.com/Sandertv/go-raknet)
by **Sander ten Veldhuis** (MIT), taken from the
[lunar-bedrock/go-raknet](https://github.com/lunar-bedrock/go-raknet/tree/lunar) fork
(branch `lunar`), which added congestion control, transport metrics and a larger
test suite. Thank you to Sander and to the lunar-bedrock contributors for the
original work. Helix maintains and changes it from here on.

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

## Contributing

See the [contributing guide](https://github.com/helix-bedrock/.github/blob/main/CONTRIBUTING.md).

## License

MIT. See [`LICENSE`](LICENSE). The original copyright notice from go-raknet is kept,
as the MIT license requires. Sources are listed in [`CREDITS.md`](CREDITS.md).
