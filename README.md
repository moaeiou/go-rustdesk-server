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

By default, clients use the configured ID-server host for relay connections on port `21117`.
If the relay is on another public host or port, set `relay_server` in `config.toml`:

```toml
relay_server = "[2001:db8::10]:21117"
```

### Configuration

The server reads `config.toml` from its working directory. If the file lives
elsewhere, set only its location, for example:
`RUSTDESK_CONFIG=/etc/rustdesk/config.toml ./go-rustdesk-server`.

| TOML key          | Default              | Purpose                                             |
| ----------------- | -------------------- | --------------------------------------------------- |
| `bind`            | `::`                 | IPv4/IPv6 listen address                            |
| `base_port`       | `21116`              | ID-server port; related services use standard offsets |
| `relay_server`    | empty                | Optional public relay override                      |
| `data_dir`        | executable directory | Keys and peer state directory                       |
| `log_path`        | executable directory | Log file path                                       |
| `log_level`       | `info`               | `info`, `warn`, or `error`                          |
| `log_color`       | `true`               | Enable terminal colors                              |
| `max_connections` | `4096`               | Maximum concurrent protocol TCP connections         |
| `max_frame_size`  | `16777216`           | Maximum framed message size                         |

Relative `data_dir` and `log_path` values are resolved from the TOML file's
directory. Unknown keys and malformed TOML stop startup with a clear error.

Peer identities and public keys are persisted atomically in `peers.json`. Logs rotate at 10 MiB and retain three backups.

```bash
./go-rustdesk-server help
```

Using this to get help

### Service Management

Auto-start is usually enabled by default. If the service doesn't start automatically, run the `enable` command manually.

| Action                | Command                                               |
| --------------------- | ----------------------------------------------------- |
| Start                 | `/etc/init.d/rsop start`                              |
| Restart               | `/etc/init.d/rsop restart`                            |
| Check status          | `/etc/init.d/rsop status`                             |
| Enable on boot        | `/etc/init.d/rsop enable`                             |
| Show Key              | `cat /etc/rustdesk/id_ed25519.pub`                    |
| Check local TCP service | `/etc/rustdesk/go-rustdesk-server doctor tcp 21116` |
| Check remote UDP service | `/etc/rustdesk/go-rustdesk-server doctor example.com udp 21116` |

## 🔨 Building from Source

### Binary

```bash
go build
```

## ⚖️ License

This project licensed under the [MoPL](https://867678.xyz/docs/mopl)

Source repository using MIT <https://github.com/danbai225/go-rustdesk-server> can change to MoPL.

We also copied: Rustdesk Server. It is licensed under the GNU AGPL Version 3 <https://www.gnu.org/licenses/agpl-3.0.html>.
