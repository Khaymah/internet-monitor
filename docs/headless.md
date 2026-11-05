# Running 24/7 on a Raspberry Pi, a home server or Docker

The strongest evidence comes from a device that never sleeps and is plugged into the
router by cable. A laptop that sleeps leaves "not monitored" gaps in the report. A
Raspberry Pi or any always-on Linux box fixes that.

The headless build is the same monitor without the tray icon or window. You use the
dashboard from a browser.

## Option 1: a Raspberry Pi or Linux machine

1. Download the file for your device from the latest release:
   - `internet-monitor-headless-linux-arm64`: Raspberry Pi 4 or 5 with 64-bit Raspberry Pi OS
   - `internet-monitor-headless-linux-armv7`: Raspberry Pi 2 or 3, or 32-bit Raspberry Pi OS
   - `internet-monitor-headless-linux-amd64`: any other Linux PC or server
2. Make it executable and start it:

   ```bash
   chmod +x internet-monitor-headless-linux-arm64
   ./internet-monitor-headless-linux-arm64 --dir ~/internet-monitor --listen 0.0.0.0
   ```

3. It prints a link with your access key, for example
   `http://192.168.1.20:8765/?session=…`. Open it from any phone or PC on your network.
   Keep the link private: anyone with it can open your dashboard.

`--listen 0.0.0.0` opens the dashboard to your home network. Without it, the dashboard
only answers on the device itself (`127.0.0.1`).

## Use it from your phone

1. Open the dashboard on your phone. The easiest way is to scan a QR code with the
   phone's camera while the phone is on the same Wi-Fi:
   - the headless build prints one under the links when it starts (read it again
     with `docker logs internet-monitor` or `journalctl -u internet-monitor`), or
   - open the dashboard on a PC and go to **Settings → App → Open on your phone**.
     This card only appears when the dashboard is open to your network.

   The code is a link with your access key in it, so don't post a photo of it.
2. In the phone's browser, choose **Add to Home screen** (Android: the ⋮ menu; iPhone:
   the Share button). You get an Internet Monitor icon that opens the dashboard.
3. The access key can change when the app restarts. If the phone then says to open the
   dashboard from the app, scan the code again.

The dashboard has no offline mode and doesn't install a service worker. Browsers only
allow service workers on https or on the device itself, and the dashboard on your
network is plain http. For the same reason, some phone browsers open the home-screen
icon as a normal browser tab instead of a full-screen app.

### Start it automatically (systemd)

Save this as `/etc/systemd/system/internet-monitor.service`, replacing `pi` with your user:

```ini
[Unit]
Description=Internet Monitor
After=network-online.target
Wants=network-online.target

[Service]
User=pi
ExecStart=/home/pi/internet-monitor-headless-linux-arm64 --dir /home/pi/internet-monitor --listen 0.0.0.0
Restart=always

[Install]
WantedBy=multi-user.target
```

Then run:

```bash
sudo systemctl enable --now internet-monitor
```

To read the dashboard link again:

```bash
journalctl -u internet-monitor | grep session=
```

## Option 2: Docker

```bash
docker compose up -d --build
docker logs internet-monitor
```

The included `docker-compose.yml` uses the host network, so the checks and the route
trace see your real connection and not Docker's internal network. Your data is kept in
`./data`. Set `TZ` in the compose file to your time zone so the report shows local
times.

## Settings

| Flag | Environment variable | Default | Meaning |
| ---- | -------------------- | ------- | ------- |
| `--dir` | `IM_DIR` | the program's folder | Where `config.json`, `case.json` and `logs/` are kept |
| `--listen` | `IM_LISTEN` | `127.0.0.1` | Address the dashboard listens on. `0.0.0.0` opens it to your network |

Everything else (targets, alerts, ntfy, Telegram, the weekly summary) is set from the
dashboard, exactly like the desktop version.

## Security

- Every request needs the access key from the link. Without it the dashboard shows
  nothing and the API answers 401.
- Requests must use the device's own address (an IP address, its host name or
  `name.local`). Requests that name another host are refused, which blocks
  DNS-rebinding attacks from web pages.
- The connection is plain http, so only open the dashboard to a home network you
  trust. Don't forward the port from your router to the internet.
