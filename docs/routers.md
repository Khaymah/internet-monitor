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
