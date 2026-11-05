# Publishing to the Microsoft Store

The Store build is the window app packaged as MSIX. Microsoft signs it on submission,
so Store users never see the SmartScreen warning.

## Status: paused

Store publishing is paused. Releases ship on GitHub only, and CI doesn't build the
MSIX. Everything below stays ready. To resume:

1. GitHub → the repository's **Settings → Secrets and variables → Actions → Variables**:
   add `STORE_BUILD` with the value `true`. Every CI run then builds the `msix` artifact
   again.
2. Finish the submission in Partner Center with `docs/store-listing.md` (text, screenshots,
   and the remaining fields).

## One-time setup

1. In Partner Center, open **Apps and games → New product → MSIX or PWA app** and
   reserve the name.
2. Open the product → **Product management → Product identity** and copy four values
   into `packaging/msix/identity.json`:

   | identity.json | Partner Center |
   | ------------- | -------------- |
   | `identity_name` | `Package/Identity/Name` |
   | `publisher` | `Package/Identity/Publisher` (starts with `CN=`) |
   | `publisher_display_name` | `Package/Properties/PublisherDisplayName` |
   | `display_name` | The reserved app name, exactly as reserved (`Package/Properties/DisplayName`) |

   They are public (they appear in every installed package), so they live in the repo.
   Until they are filled in, CI skips the MSIX step. The Store rejects the package if
   `display_name` differs from the reserved name in any way, even a space for a hyphen.

## Each Store release

1. Merge a final (non-rc) version. With `STORE_BUILD` set, CI builds
   `internet-monitor-store.msix` as the `msix` artifact of the run (Actions → the run →
   Artifacts). It isn't attached to the GitHub release: an unsigned MSIX can't be
   installed outside the Store.
2. In Partner Center, create a submission and upload the `.msix` under **Packages**.
   The package version is the release version with a `.0` revision (`2.0.4` → `2.0.4.0`),
   and the Store requires it to be higher than the last one you submitted. So submit
   final versions only, not rcs.
3. **Restricted capability `runFullTrust`.** The Store asks why the app needs it. A
   suitable answer: "Internet Monitor is a desktop app (Win32) that runs a local
   monitoring service, shows a tray icon, and runs the system's tracert and netsh
   tools to diagnose outages."
4. **Privacy policy URL:** `https://github.com/Khaymah/internet-monitor/blob/master/PRIVACY.md`.
   It's the same text as the in-app Privacy page (`web/locales/*/legal.json`), in English and
   Arabic. The in-app page itself can't be used, since it's only served on 127.0.0.1. The
   link works once `PRIVACY.md` is on `master`; when the app's data or connections change,
   update both and the date.

## What changes in the Store build

The same executables are used. When Windows runs them from a package, the app detects
it and changes four things:

| Area | Normal build | Store build |
| ---- | ------------ | ----------- |
| Data (`config.json`, `case.json`, `logs/`) | Next to the program | `%LOCALAPPDATA%\InternetMonitor` |
| Updates | Built-in updater from GitHub | The Store updates the app; the updater is off |
| Start with Windows | Settings toggle (registry) | Declared in the package; the user turns it on or off in Windows Settings → Apps → Startup |
| Notifications | Own app ID registered in the registry | The package's app ID |

At sign-in Windows starts the tray build in the background, so no window pops up.
Opening Internet Monitor from the Start menu while it already runs opens the dashboard.
