#!/usr/bin/env bash
#
# Prints the live-test groups relevant to this PR's diff, as ONE line:
#   NONE          no provider code changed -> skip live tests
#   all           a shared or unrecognized change -> run everything
#   kafka,flink   run just these groups
#
# It maps each changed resource / data source to its group by name (the table
# below). Anything it doesn't recognize -- a shared file, or a resource we
# haven't listed -- runs everything, so it never skips a test it should run.
# You only touch the table when a brand-new live-test GROUP is added.
#
# Env: BASE_REF (default origin/master), CHANGED (override the diff, for tests).

set -euo pipefail
BASE_REF="${BASE_REF:-origin/master}"
DIR="internal/provider"

# resource/data-source filename -> its live-test group.
group_for() {
  case "$1" in
    *flink*)                                                     echo flink ;;
    *kafka*|*_acl*|*cluster_link*|*mirror*|*client_quota*|*consumer_group*) echo kafka ;;
    *schema*|*subject*|*exporter*|*kek*|*dek*)                   echo schema_registry ;;
    *connect*|*plugin*)                                          echo connect ;;
    *network*|*private_link*|*peering*|*dns*|*gateway*|*access_point*|*transit*|*egress*|*ip_filter*|*ip_group*) echo networking ;;
    *catalog*|*business_metadata*|*_tag*|*entity*)               echo data_catalog ;;
    *role_binding*|*identity*|*group_mapping*|*invitation*)      echo rbac ;;
    *tableflow*)                                                 echo tableflow ;;
    *environment*|*organization*|*service_account*|*api_key*|*byok*|*provider_integration*|*rtce*|*ksql*|*user*|*certificate*) echo core ;;
    *)                                                           echo all ;;   # unknown -> safe
  esac
}

changed="${CHANGED:-}"
if [ -z "$changed" ]; then
  git fetch -q origin "${BASE_REF#origin/}" 2>/dev/null || true
  base=$(git merge-base HEAD "$BASE_REF" 2>/dev/null || echo "$BASE_REF")
  changed=$(git diff --name-only "$base" HEAD 2>/dev/null || true)
fi
[ -n "${changed// }" ] || { echo NONE; exit 0; }

groups=""
add() { case ",$groups," in *",$1,"*) ;; *) groups="${groups:+$groups,}$1";; esac; }

touched=0
while IFS= read -r f; do
  [ -n "$f" ] || continue
  case "$f" in
    go.mod|go.sum)                                echo all; exit 0 ;;   # dep bump -> everything
    "$DIR"/resource_*.go|"$DIR"/data_source_*.go)
      touched=1
      g=$(group_for "$(basename "$f")")
      [ "$g" = all ] && { echo all; exit 0; }
      add "$g" ;;
    "$DIR"/*.go)                                  echo all; exit 0 ;;   # shared provider file
    *) : ;;                                                             # docs, examples, ci -> ignore
  esac
done <<EOF
$changed
EOF

if [ -n "$groups" ]; then echo "$groups"
elif [ "$touched" = 1 ]; then echo all
else echo NONE
fi
