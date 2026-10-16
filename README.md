# go-rustdesk-server

A RustDesk server rebuilt using Golang

## ⬇️ Download

[GitHub Releases](https://github.com/moaeiou/go-rustdesk-server/releases)

## ✨ Features

- 🩺 **Designed to cure the high blood pressure caused by compiling the original RustDesk Server**
- 🚫 Docker support, api_server, and WebUI have been completely removed
- ⚡ Pure, lightweight, and ready to run out of the box

## ⚠️ Before You Start

- **TCP and UDP ports 21114–21119 must all be open**, or the server will not work correctly.
- Since this project has no WebUI, port `21114` and `21119` hosts a stub WebServer that always returns `200 OK` to prevent client-side errors.

## 🚀 Quick Start

### ✈️ RUN

```bash
./go-rustdesk-server
```

That's it. The server will start listening on TCP/UDP ports **21114–21119**.  
Configure your RustDesk client just as you would with a standard rustdesk-server setup.

If you need more custom settings , please create `config.toml`.
For more information about the `config.toml`, see [Configuration](#configuration)

### ⚙️ RUN as a service

The repository includes ready-to-use service definitions under `daemon/`. Both
examples start the server with an explicit configuration file and restart it
after an unexpected failure.

#### systemd

The supplied unit runs as an unprivileged `rustdesk` user. Its expected paths
are:

- executable: `/usr/local/bin/go-rustdesk-server`
- configuration: `/etc/go-rustdesk-server/config.toml`
- state and keys: `/var/lib/rustdesk`
- logs: `/var/log/rustdesk`

Set these paths in the installed configuration:

```toml
[base]
data_dir = "/var/lib/rustdesk"

[log]
log_path = "/var/log/rustdesk/rdks.log"
```

Install and start the service:

```bash
sudo install -m 0755 go-rustdesk-server /usr/local/bin/go-rustdesk-server
sudo install -d -m 0755 /etc/go-rustdesk-server
sudo install -m 0644 config.toml /etc/go-rustdesk-server/config.toml
sudo useradd --system --home-dir /var/lib/rustdesk --shell /usr/sbin/nologin rustdesk 2>/dev/null || true
sudo install -m 0644 daemon/systemd/go-rustdesk-server.service /etc/systemd/system/go-rustdesk-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now go-rustdesk-server
```

Useful commands:

```bash
systemctl status go-rustdesk-server
journalctl -u go-rustdesk-server -f
sudo systemctl restart go-rustdesk-server
```

#### OpenWrt / ImmortalWrt init.d

If you prefer a LuCI interface, see [LuCI-APP-RSOP](https://github.com/morouter/luci-app-rsop).

The supplied script is a native `procd` service with automatic respawn. It
expects:

- executable: `/usr/bin/go-rustdesk-server`
- configuration: `/etc/go-rustdesk-server/config.toml`

Install and start it:

```sh
cp go-rustdesk-server /usr/bin/go-rustdesk-server
chmod 0755 /usr/bin/go-rustdesk-server
mkdir -p /etc/go-rustdesk-server
cp config.toml /etc/go-rustdesk-server/config.toml
chmod 0644 /etc/go-rustdesk-server/config.toml
cp daemon/openwrt/go-rustdesk-server.init /etc/init.d/go-rustdesk-server
chmod 0755 /etc/init.d/go-rustdesk-server
/etc/init.d/go-rustdesk-server enable
/etc/init.d/go-rustdesk-server start
```

Check status and follow logs:

```sh
/etc/init.d/go-rustdesk-server status
logread -f | grep go-rustdesk-server
```

After editing `config.toml`, apply it with:

```sh
/etc/init.d/go-rustdesk-server reload
```

### 🔨 How to build

```bash
go build
```

### Configuration

Start the server with an explicit configuration file:

```bash
./go-rustdesk-server run -c ./config.toml
```

Both `-c` and `--config` are supported. Relative paths inside the configuration
are resolved from the configuration file's directory. Running the binary
without a command remains supported and reads `config.toml` from the working
directory.

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
| `extra.enforce_key`     | `true`               | Reject clients with a missing or wrong key  |
| `extra.relay_buffer_size` | `262144`            | Per-socket Relay TCP buffer; `0` uses OS default |

Relative `base.data_dir` and `log.log_path` values are resolved from the TOML
file's directory. Unknown keys and malformed TOML stop startup with a clear
error.

With `extra.enforce_key = true`, punch-hole and relay requests must carry the
public key printed by the server (the contents of `id_ed25519.pub`). Set it to
`false` only when intentionally allowing clients without the correct key; they
may show an insecure-connection confirmation instead.

`extra.relay_buffer_size` helps absorb short traffic bursts on unstable links.
The default 256 KiB balances jitter tolerance and memory use on routers. Values
from 64 KiB through 16 MiB are accepted; use `0` to retain operating-system TCP
autotuning. Larger values consume more kernel memory per active Relay client
and cannot eliminate TCP head-of-line blocking caused by packet loss.

Peer identities and public keys are persisted atomically in `peers.toml`; no
peer-store backup file is created. Existing `peers.json` data is migrated once
and removed after a successful TOML write. Logs rotate at `log.max_size`; the
oldest file is destroyed after `log.max_files` is exceeded. Set `max_files = 1`
to retain no log history.

```bash
./go-rustdesk-server help
```

Using this to get help

## ⚖️ License

This project licensed under the [MoPL](https://867678.xyz/docs/mopl)

Source repository using MIT <https://github.com/danbai225/go-rustdesk-server> so here change to MoPL.

We also copied: Rustdesk Server. It is licensed under the GNU AGPL Version 3 <https://www.gnu.org/licenses/agpl-3.0.html>
