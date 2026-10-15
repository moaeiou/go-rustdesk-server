# go-rustdesk-server

Forked from [danbai225/go-rustdesk-server](https://github.com/danbai225/go-rustdesk-server).

A RustDesk server rebuilt using Golang

## ⬇️ Download

[GitHub Releases](https://github.com/moaeiou/go-rustdesk-server/releases)

## ✨ Features

- 🩺 **Designed to cure the high blood pressure caused by compiling the original RustDesk Server**
- 🚫 Docker support, api_server, and WebUI have been completely removed
- ⚡ Pure, lightweight, and ready to run out of the box
- 📦 OpenWrt support included (rsop project merged)

## ⚠️ Before You Start

- **TCP and UDP ports 21114–21119 must all be open**, or the server will not work correctly.
- Since this project has no WebUI, port `21114` and `21119` hosts a stub WebServer that always returns `200 OK` to prevent client-side errors.

## 🚀 Quick Start

```bash
./go-rustdesk-server
```

That's it. The server will start listening on TCP/UDP ports **21114–21119**.  
Configure your RustDesk client just as you would with a standard rustdesk-server setup.

By default, the server derives the relay address from the connection-facing
interface and the route to the other peer. If the server is behind NAT, or the
relay uses another public host or port, set `base.relay_server` in `config.toml`:

```toml
[base]
relay_server = "[2001:db8::10]:21117"
```

### Configuration

The server reads `config.toml` from its working directory. If the file lives
elsewhere, set only its location, for example:
`RUSTDESK_CONFIG=/etc/rustdesk/config.toml ./go-rustdesk-server`.

| TOML key                | Default              | Purpose                                     |
| ----------------------- | -------------------- | ------------------------------------------- |
| `base.listen`           | `::`                 | IPv4/IPv6 listen address                    |
| `base.http_port`        | `21114`              | HTTP compatibility service                  |
| `base.nat_port`         | `21115`              | NAT test service                            |
| `base.id_port`          | `21116`              | ID server on both TCP and UDP               |
| `base.relay_port`       | `21117`              | Relay service                               |
| `base.websocket_port`   | `21118`              | WebSocket service                           |
| `base.web_api_port`     | `21119`              | HTTP compatibility/Web API service          |
| `base.relay_server`     | empty                | Optional public relay override              |
| `base.data_dir`         | config directory     | Keys and peer state directory               |
| `log.log_path`          | executable directory | Log file path                               |
| `log.log_level`         | `info`               | `info`, `warn`, or `error`                  |
| `log.log_color`         | `true`               | Enable terminal colors                      |
| `log.max_size`          | `10485760`           | Maximum bytes in one log file               |
| `log.max_files`         | `4`                  | Total retained files, including current     |
| `extra.max_connections` | `4096`               | Maximum concurrent protocol TCP connections |
| `extra.max_frame_size`  | `16777216`           | Maximum framed message size                 |

Relative `base.data_dir` and `log.log_path` values are resolved from the TOML
file's directory. Unknown keys and malformed TOML stop startup with a clear
error.

Peer identities and public keys are persisted atomically in `peers.toml`; no
peer-store backup file is created. Existing `peers.json` data is migrated once
and removed after a successful TOML write. Logs rotate at `log.max_size`; the
oldest file is destroyed after `log.max_files` is exceeded. Set `max_files = 1`
to retain no log history.

```bash
./go-rustdesk-server help
```

Using this to get help

### Service Management

Auto-start is usually enabled by default. If the service doesn't start automatically, run the `enable` command manually.

| Action                      | Command                                                         |
| --------------------------- | --------------------------------------------------------------- |
| Start                       | `/etc/init.d/rsop start`                                        |
| Restart                     | `/etc/init.d/rsop restart`                                      |
| Check status                | `/etc/init.d/rsop status`                                       |
| Enable on boot              | `/etc/init.d/rsop enable`                                       |
| Show Key                    | `cat /etc/rustdesk/id_ed25519.pub`                              |
| Check all standard services | `/etc/rustdesk/go-rustdesk-server doctor example.com`           |
| Check one service           | `/etc/rustdesk/go-rustdesk-server doctor example.com:21116 udp` |

Doctor prints every address returned by the system resolver and then tests each
resolved IPv4 and IPv6 address independently without replacing or filtering it.

## 🔨 How to build

```bash
go build
```

## ⚖️ License

This project licensed under the [MoPL](https://867678.xyz/docs/mopl)

Source repository using MIT <https://github.com/danbai225/go-rustdesk-server> can change to MoPL.

We also copied: Rustdesk Server. It is licensed under the GNU AGPL Version 3 <https://www.gnu.org/licenses/agpl-3.0.html>.
