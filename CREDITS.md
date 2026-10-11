# Credits

Code in this repo that was ported from other projects, as required by the
[porting rules](https://github.com/helix-bedrock/.github/blob/main/PORTING.md).

| Helix path | Source project | Source path | Commit | License |
|---|---|---|---|---|
| Entire repository (initial import) | [go-raknet](https://github.com/Sandertv/go-raknet) by Sander ten Veldhuis, via a fork with congestion control and metrics | All files | Fork snapshot (working branch `lunar`), imported 2026-10-11 | MIT |
| Retransmission timing and initial congestion window | [lunar-bedrock/go-raknet#37](https://github.com/lunar-bedrock/go-raknet/pull/37) by @HashimTheArab | `congestion.go`, `resend_map.go` and tests | PR head at import | MIT |
| Ignoring offline replies with invalid magic | [lunar-bedrock/go-raknet#66](https://github.com/lunar-bedrock/go-raknet/pull/66) by @JustTalDevelops | `dial.go`, `internal/message` and tests | PR head at import | MIT |
| Final MTU from Open Connection Reply 2 | [lunar-bedrock/go-raknet#67](https://github.com/lunar-bedrock/go-raknet/pull/67) by @JustTalDevelops | `dial.go` and tests | PR head at import | MIT |
