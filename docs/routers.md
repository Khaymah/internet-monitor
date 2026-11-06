# Supported routers

Internet Monitor reads the home router in two layers. They are independent, and the result is merged into one sample per minute.

1. **UPnP IGD** (always, no password): link state, sync speed, connection state, uptime, last disconnect reason and traffic counters, when the router offers them. See `internal/router/router.go`.
2. **A driver** (optional): reads the router's own management interface for what UPnP does not give (noise margin, attenuation, line uptime, reliable counters). Drivers live in `internal/router/` and are picked automatically.

The app never needs to be told which driver to use. The detected driver name is shown on the router card and in Settings → App, and is returned as `driver` by `GET /api/router` and `POST /api/test-router`.

## Drivers

| Driver | Name in the API | Needs a password | Reads |
| --- | --- | --- | --- |
| Huawei home gateways | `huawei-hg` | yes | `/api/ntwk/dslinfo` (rates, SNR, attenuation, line uptime), `/api/ntwk/wan_st` (byte counters), `/api/system/deviceinfo` (model) |
| TR-064 | `tr064` | only if the router asks | WAN counters, DSL figures, connection status, uptime, last error, model |

### Huawei family (`huawei-hg`)

Covers the HG630 V2 (tested against a fake of its real responses), and the HG633, HG659, HG658 and DG8045 style gateways that share the same web API: a `GET /` page with `csrf_param` and `csrf_token` meta tags, `POST /api/system/user_login` with `sha256(user + base64(hex(sha256(password))) + csrf_param + csrf_token)`, and `while(1); /*…*/` wrapped JSON.

- Detection: the CSRF meta tags on `/` or on `/html/index.html` (some firmware serves them only there), or a `/api/system/deviceinfo` answer with `DeviceName` or `SerialNumber`.
- Gateways without a DSL page (GPON/cable models, where `/api/ntwk/dslinfo` answers 404) are still read: the counters from `/api/ntwk/wan_st` and the model are used, and the line figures are left out. `wan_st` may be a list (summed) or one object.
- It signs in, reads, and signs out on every read.

Sources used for the endpoints and the hash: the HG659 write-up at engineered.at/articles/hacking-the-hg659, the HG633 reboot gist by RazZziel (`/api/system/deviceinfo` `UpTime`, `/api/ntwk/dslinfo` `Status`, `DownCurrRate`, `UpCurrRate`, `DownstreamMaxBitRate`, `UpstreamMaxBitRate`), the WS319 gist by gmikhail (CSRF values also read from `/html/index.html`), `axatol/go-huawei-hg659` (device info field names: `DeviceName`, `SerialNumber`, `UpTime`) and `wbmeehan/huawei-hg659-api` (`/api/system/user_login`, `/api/system/user_logout`).

Not yet confirmed on real hardware: the exact `wan_st` shape of non-HG630 models and whether newer firmware changes the login hash.

### TR-064 (`tr064`)

A standard (Broadband Forum) used by FRITZ!Box and by some other DSL and fiber routers. The description is at `/tr64desc.xml` on port 49000 (http) or 49443 (https, self-signed).

- Detection: the description loads and lists `urn:dslforum-org:service:*` services whose control URLs are on the router itself.
- Reads, each only if the router offers the service:
  - `WANCommonInterfaceConfig`: `GetAddonInfos` (64-bit `NewX_AVM_DE_TotalBytesReceived64` / `…Sent64` on FRITZ!Box), else `GetTotalBytesReceived` / `GetTotalBytesSent` (32-bit, the usage code handles the wrap).
  - `WANDSLInterfaceConfig:GetInfo`: status, current and maximum rates (kbit/s), noise margin and attenuation (0.1 dB units, divided by 10).
  - `WANPPPConnection` then `WANIPConnection:GetStatusInfo`: connection status, uptime, last connection error.
  - `DeviceInfo:GetInfo`: model, and device uptime when the connection gives none.
- Authentication is HTTP digest (MD5 and SHA-256, `qop=auth`). The app first asks without credentials; only on a `401` challenge does it answer once with the saved user and password. With no password saved, a router that insists on one is left alone (no error is shown, because the user never asked for it).
- Routers without a DSL service (fiber, cable) work: they give counters and connection state, and the card simply shows no line figures.
- `X_AVM-DE_GetOnlineMonitor` is not used: it returns per-second throughput series, not totals, and `GetAddonInfos` already has the 64-bit totals.

## How detection works

`Poller` (`poller_web.go`) keeps the chosen driver between reads.

1. A `Probe` is created for the router host. It holds one HTTP client (private addresses only, legacy TLS allowed for this host only, redirects only to the same host) and caches the first page (`GET /`, https then http), so the fingerprint costs one request however many drivers look at it.
2. `Detect` runs each driver's `Detect` in order (`Drivers()` in `driver.go`). The first that answers true wins. With no password saved, only drivers that implement `Anonymous() bool` are tried.
3. If none matches, the router is reported as unsupported (`web_error: "unsupported"`) and not tried again until the address, user or password changes. If the router did not answer at all, that is reported as unreachable and retried.
4. The driver is cached per address. It is dropped, and detection runs again, after 3 failed reads in a row (a rejected password or a busy router does not count), when the address changes, when a password is added, or on a Test.
5. Without a saved password and without an explicit address, nothing is probed unless UPnP found the router, so the app never knocks on an address it only guessed.

If a page read fails while the previous one supplied byte counters, the UPnP counters are dropped from that sample, so two counters that may count differently (for example 32-bit and 64-bit) are never mixed in the usage maths.

## Safety rules every driver must keep

- Private addresses only. `NewProbe` refuses anything else, and `Probe.Get` and the TR-064 session refuse URLs on another host.
- Legacy TLS and unverified certificates are allowed only through the probe's client, which only talks to the router host.
- A rejected password is returned as `ErrWrongPassword`, and the poller never tries the same address, user and password again. A driver must not retry a login inside one read.
- A busy router is `ErrRouterBusy`. An unreadable page is `ErrWebUnknown`.
- The poller sets the request rate: one read every 5 minutes at most (sooner only on fast lines so a 32-bit counter wraps at most once).
- If the router has sessions, sign out at the end of every read.
- Never write the password to a log or an error.

## Adding a driver

1. Create `internal/router/<name>.go` with a type that implements:

   ```go
   type Driver interface {
       Name() string
       Detect(ctx context.Context, p *Probe) (bool, error)
       Read(ctx context.Context, p *Probe, c Creds) (WebStats, error)
   }
   ```

   `Detect` must be cheap and must not send credentials: use `p.Page(ctx)` (cached) or one extra `p.Get`. Return `false, nil` for "not mine". `Read` fills `WebStats` (`HasLineStats` and `HasCounters` say which parts are real) and returns the sentinel errors above. Add `Anonymous() bool` returning true only if the driver can work with no password.
2. Add it to `Drivers()` in `driver.go`. Put a vendor-specific driver before a generic one such as TR-064.
3. Add the name to `DRIVER_LABEL` in `web/src/components/RouterReading.tsx` and a `rt_driver_<name>` string to `web/locales/<lang>/overview.json` in all five languages (until then the card says "Reading the router's own page").
4. Write a fake server test (see `huawei_test.go`, `huawei_family_test.go`, `tr064_test.go`). It must cover: a good read with real field names, a wrong password tried once, a router that only has some of the data, and an unrelated page that is not detected. Use the real login hash or digest computed independently in the test, not the driver's own function, where you can.
5. Document it in this file, with the public sources for the endpoints.

## Not supported yet

- **Huawei EchoLife fiber ONTs** (HG8145V5, HG8245H, EG8145…). They sign in through `/asp/GetRandCount.asp` and `/login.cgi` (base64 password, token bound to the TCP connection, cookies). Public sources describe that login and a device-list page, but none documents a stats page with byte counters or optical power in enough detail to build a faithful fake server, so no driver was written. A driver written blind would be untested against any real response.
- ZTE and TP-Link drivers are planned on top of the same interface.

Not on the list? Open an issue with your model: https://github.com/Khaymah/internet-monitor/issues/new?labels=router-support
## Asking for your router to be supported

Internet Monitor reads your router to say whose fault an outage was and to track your data plan. Today it reads any router that offers UPnP, plus the Huawei HG630 V2 web page. We can add more routers (ZTE, other Huawei models, TP-Link, TR-064 boxes), but we don't own them. The app can build a report that tells us how your router answers, so we can write a driver for it.

### What to do

1. Open Settings, then App, and turn on "Read the router's status".
2. Find "Help us support your router" and press "Create report". It takes up to a minute.
3. Read the summary and, if you like, the full report. Nothing has left your computer yet.
4. Press "Download report". You get `router-report-<model>.json`.
5. Press "Open a GitHub issue", drag the downloaded file into the box, and submit. You can also open a "Router support" issue from the Issues tab of github.com/Khaymah/internet-monitor.

The app never uploads the report. You send it, through GitHub, and only if you submit the issue.

### What the report contains

- The app version, your operating system and its architecture.
- The UPnP description of your router: maker, model, software version, the services and actions it lists, and the shape of the answers to the status actions the app uses. Numbers such as speeds, counters and uptime are kept.
- Whether the TR-064 description (`http://<router>:49000/tr64desc.xml`) exists, with its model fields and service list.
- A fingerprint of the router's web page on ports 80 and 443: status code, `Server` header, page title, the names (never the values) of login form fields, script addresses, the paths those scripts mention, and a short list of brand names found in the page. No sign-in happens for this part.

### The optional sign-in probe

If you tick "Also try signing in with my saved router password", and the page looks like a family the app can already sign in to (today only Huawei), the app signs in once with the password you saved, reads a few status pages and records only the names and kinds of the fields they return, never their values. It signs out afterwards. If the router rejects the password, the app does not try again, and it stops using that password until you change it.

### What is removed

Every report is cleaned in code before you see it. These never appear:

- MAC addresses, serial numbers, IMEI and IMSI numbers
- Wi-Fi names and passwords
- public IP addresses (the report only says whether an address is private)
- user names, passwords, tokens, session IDs, cookies and CSRF values
- phone numbers, PPPoE user names and email addresses
- names of the devices on your network
- long hex and base64 strings that look like secrets

Only addresses inside your home network are contacted while building the report.

### For driver authors

A driver can take part in the sign-in probe by implementing `router.DeepProber` (`Family`, `Matches` and `Probe`) and registering it with `router.RegisterDeepProber`. `Matches` receives the web page fingerprints, `Probe` signs in with the saved credentials and returns the endpoints it read as shapes built with the same cleaning rules. The report code lives in `internal/router/report*.go`, and the API is `POST /api/router/report`.
