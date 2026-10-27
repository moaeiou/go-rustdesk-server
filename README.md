# go-rustdesk-server

A RustDesk server rebuilt using Golang

## ⬇️ Download

[GitHub Releases](https://github.com/moaeiou/go-rustdesk-server/releases)

## 🎉 This project has been archived!

The rustdesk-server has more problem

Such as:
- Slowly connection
- If even a single packet fails to arrive, everything comes to a complete standstill.
- The sound remains normal when it freezes.
- The encoding and decoding speeds are far too slow, even when hardware encoding/decoding is enabled on both ends.

But now I have OpenUUYC, which resolves many of the issues that UUYC had on Linux.

So, I concluded that there was no need to continue maintaining it.

## ✨ Features

- 🩺 **Designed to cure the high blood pressure caused by compiling the original RustDesk Server**
- 🚫 Docker support, api_server, and WebUI have been completely removed
- ⚡ Pure, lightweight, and ready to run out of the box

## ⚠️ Before You Start

- Native clients use TCP `21115–21117` and UDP `21116`.
- Web clients additionally use WebSocket TCP `21118` (signalling) and `21119`
  (relay). TLS/WSS can be terminated at a reverse proxy.
- `21114` remains an HTTP compatibility stub. Ordinary HTTP requests on `21119`
  also return `200 OK`; WebSocket upgrades now enter the real relay. Neither
  endpoint implements accounts, an address book, or a management WebUI.

## 🚀 Quick Start

### ✈️ RUN

```bash
./go-rustdesk-server
```

That's it. The server listens on TCP **21114–21119** and UDP **21116**.
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
| `base.web_api_port`     | `21119`              | WebSocket relay + HTTP compatibility          |
| `base.relay_servers`    | `[]`                 | One relay endpoint as a string, or multiple endpoints as an array |
| `base.config_serial`    | `0`                  | Legacy configuration revision |
| `base.rendezvous_servers` | `[]`               | ID servers distributed to older clients |
| `extra.always_use_relay` | `false`              | Disable direct and LAN connection negotiation |
| `extra.udp_receive_buffer` | `0`                | UDP receive buffer in bytes; 0 uses OS default |
| `extra.single_bandwidth` | `0`                 | Per-session bidirectional bandwidth, decimal Mb/s; 0 unlimited |
| `extra.total_bandwidth` | `0`                  | Aggregate relay bandwidth, decimal Mb/s; 0 unlimited |
| `extra.blocklist`       | `[]`                 | IPs/CIDRs refused by the relay |
| `extra.blacklist`       | `[]`                 | IPs/CIDRs limited to limit_speed |
| `extra.limit_speed`     | `32`                 | Blacklist/downgraded-session bandwidth, Mb/s |
| `extra.downgrade_threshold` | `0.66`           | Lifetime-average fraction of single_bandwidth triggering downgrade |
| `extra.downgrade_start_check` | `1800`         | Seconds before downgrade checks; 0 disables |
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
peer-store backup file is created. Logs rotate at `log.max_size`; the oldest
file is destroyed after `log.max_files` is exceeded. Set `max_files = 1` to
retain no log history.

### Additional server features

Native TCP relay remains a byte stream. WebSocket relay preserves RustDesk
message boundaries and supports TCP↔WebSocket and WebSocket↔WebSocket pairs.
WebSocket requests use the same server-key validation as TCP requests. Browser
origins are allowed; forwarded IP headers are not trusted, so behind a reverse
proxy IP-based policies see the proxy's address.

For one relay server, `base.relay_servers` accepts a string such as
`"relay-a.example:21117"`. For multiple relay servers, use an array such as
`["relay-a.example:21117", "relay-b.example:21117"]`. Array selection is
round-robin, without health probing. Both clients must be able to reach the
selected server; use a public DNS name behind NAT. Web clients use the
corresponding relay's WebSocket port; the stock client convention is native
relay port + 2.

When `base.relay_servers` is empty, the server derives a relay address from the
incoming connection and route to the other peer. Behind NAT, configure a
publicly reachable endpoint in `base.relay_servers`.

Bandwidth limits apply to the sum of both directions of a session. The total
limit is shared by all sessions. Either peer matching `blacklist` applies the
reduced cap to that pair. Automatic downgrade is sticky until disconnection,
and requires a nonzero `single_bandwidth` and `downgrade_start_check`. The token
bucket allows a small 32 KiB burst. Defaults preserve this project's previous
unlimited throughput; upstream defaults are 128/1024 Mb/s.

Configuration changes take effect after service restart/reload. The server does
not expose the upstream plaintext runtime management console. See
[the upstream comparison](docs/upstream-comparison.md) for implemented features
and remaining differences.

Generate a separate key pair or validate an existing one without changing the
server's stored identity:

```bash
./go-rustdesk-server genkeypair
./go-rustdesk-server validatekeypair '<public-key>' '<secret-key>'
./go-rustdesk-server help
```

Using this to get help

## ⚖️ License

This project licensed under the [MoPL](https://867678.xyz/docs/mopl)

Source repository using MIT <https://github.com/danbai225/go-rustdesk-server> so here change to MoPL.

We also copied: Rustdesk Server. It is licensed under the GNU AGPL Version 3 <https://www.gnu.org/licenses/agpl-3.0.html>
