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
settings=$(printf '%s\n' "$changed" | grep -E '^(web/src/screens/settings/|web/locales/en/(settings|notifications)\.json$)' || true)
tutorial=$(printf '%s\n' "$changed" | grep -E '^(tools/tutorial/|docs/tutorial/|web/public/tutorial/)' || true)

need_site=""
need_tutorial=""
[ -z "$site" ] && need_site=1
[ -n "$settings" ] && [ -z "$tutorial" ] && need_tutorial=1
[ -n "$need_site$need_tutorial" ] || exit 0

stamp="$(git rev-parse --git-dir)/site-sync-reminded"
key=$(printf '%s\n%s' "$need_site$need_tutorial" "$app" | sha256sum | cut -c1-16)
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
if [ -n "$need_tutorial" ]; then
  reason+="
- Tutorial: the Settings screens changed. Update the scenes in tools/tutorial/ (narration and the 4 subtitle translations) and rebuild the video and web/public/tutorial/*.vtt (see tools/tutorial/README.md)."
fi
reason+="
If nothing needs changing, say so in one line and stop; this reminder won't repeat for the same changes."

jq -n --arg r "$reason" '{decision: "block", reason: $r}'
