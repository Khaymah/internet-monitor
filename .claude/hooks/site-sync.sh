#!/usr/bin/env bash
set -euo pipefail

input=$(cat)
if [ "$(printf '%s' "$input" | jq -r '.stop_hook_active // false')" = "true" ]; then
  exit 0
fi

root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$root"
[ -d site ] || exit 0

base=$(git merge-base HEAD origin/master 2>/dev/null || git merge-base HEAD master 2>/dev/null || true)
[ -n "$base" ] || exit 0

changed=$( {
  git diff --name-only "$base" HEAD
  git diff --name-only HEAD
  git ls-files --others --exclude-standard
} | sort -u )
[ -n "$changed" ] || exit 0

app=$(printf '%s\n' "$changed" | grep -E '^(web/src/(screens|components)/|web/locales/en/|internal/server/[^/]+_api\.go$|docs/releases/|PRIVACY\.md$)' || true)
[ -n "$app" ] || exit 0

site=$(printf '%s\n' "$changed" | grep -E '^site/' || true)

tours_of() {
  case "$1" in
    web/src/screens/Overview.tsx | web/src/components/OutageTimeline.tsx | web/src/components/RouterCard.tsx | \
      web/src/components/LatencyChart.tsx | web/locales/en/overview.json) echo overview ;;
    web/src/components/InsightsCard.tsx) printf '%s\n' overview complaints ;;
    web/src/screens/History.tsx | web/locales/en/history.json) echo history ;;
    web/src/screens/Notifications.tsx) echo notifications ;;
    web/locales/en/notifications.json) printf '%s\n' notifications settings ;;
    web/src/screens/Speed.tsx | web/src/components/BufferbloatCard.tsx | web/src/components/Gauge.tsx | \
      web/locales/en/speed.json) echo speed ;;
    web/src/screens/complaints/* | web/locales/en/complaints.json) echo complaints ;;
    web/src/screens/Report.tsx | web/src/screens/ReportSections.tsx | web/locales/en/report.json) echo report ;;
    web/src/screens/settings/* | web/locales/en/settings.json) echo settings ;;
  esac
}

stale=""
for id in $(printf '%s\n' "$app" | while read -r f; do tours_of "$f"; done | sort -u); do
  if ! printf '%s\n' "$changed" | grep -qE "^(tools/tutorial/videos/$id\.json|docs/tutorial/$id\.mp4|web/public/tutorial/$id/)"; then
    stale+="${stale:+, }$id"
  fi
done

need_site=""
[ -z "$site" ] && need_site=1
[ -n "$need_site$stale" ] || exit 0

stamp="$(git rev-parse --git-dir)/site-sync-reminded"
key=$(printf '%s\n%s\n%s' "$need_site" "$stale" "$app" | sha256sum | cut -c1-16)
if [ -f "$stamp" ] && grep -qx "$key" "$stamp"; then
  exit 0
fi
echo "$key" >> "$stamp"

files=$(printf '%s\n' "$app" | head -15 | sed 's/^/  - /')
reason="User-facing app files changed on this branch:
$files
"
if [ -n "$need_site" ]; then
  reason+="
- Website: check whether site/ (copy in site/locales/{en,ar,fr,tr,ur}.json, pages, screenshots) still describes the app correctly, and update it in all 5 languages if not. Release data, FAQ and privacy are read at build time, so only hand-written copy needs checking. Run the site build and its link check (see docs/website.md)."
fi
if [ -n "$stale" ]; then
  reason+="
- Video tours: these pages changed but their videos did not: $stale. Update the scenes in tools/tutorial/videos/<id>.json (narration and the 4 subtitle translations) and rebuild only those videos with: node tools/tutorial/build.cjs --url ... --voice ... <id> (see tools/tutorial/README.md)."
fi
reason+="
If nothing needs changing, say so in one line and stop; this reminder won't repeat for the same changes."

jq -n --arg r "$reason" '{decision: "block", reason: $r}'
