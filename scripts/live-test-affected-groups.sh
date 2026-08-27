#!/usr/bin/env bash
#
# scripts/live-test-affected-groups.sh
#
# Prints the TF_LIVE_TEST_GROUPS value for the live tests relevant to this
# branch's diff against the base branch.
#
# Resolution order for each changed internal/provider/*.go file:
#   1. stem match  -> read the group from a matching *_live_test.go build tag
#      (tries the file's own stem, its resource<->data_source sibling, and a
#       singular form, so a data source inherits its resource's live test).
#   2. keyword map -> for files with no matching live test yet (e.g. a brand-new
#      resource, or one whose live test is named differently), map by domain
#      keyword so scoping still works.
#   3. otherwise    -> blast radius (shared helper, or unrecognized) => run all.
#
# This never *under*-runs: anything unrecognized falls back to "all". The nightly
# full run remains the backstop for cross-group blast radius.
#
# Output on stdout is EXACTLY one line, one of:
#   NONE            no provider code changed -> caller should skip live tests
#   all             a shared / unrecognized file changed -> run everything
#   kafka,flink     comma-separated affected groups
#
# Diagnostics go to stderr so stdout stays machine-parseable.
#
# Env knobs:
#   BASE_REF               base to diff against            (default: origin/master)
#   PROVIDER_DIR           provider package dir            (default: internal/provider)
#   BLAST_RADIUS_GROUPS    what to run for a shared change (default: all)
#   CHANGED_FILES_OVERRIDE newline/space list, bypasses git (for tests/dry-runs)

set -euo pipefail

BASE_REF="${BASE_REF:-origin/master}"
PROVIDER_DIR="${PROVIDER_DIR:-internal/provider}"
BLAST_RADIUS_GROUPS="${BLAST_RADIUS_GROUPS:-all}"

log() { echo "[affected-groups] $*" >&2; }

# "//go:build live_test && (all || kafka || core)" -> lines: kafka, core
extract_groups() {
  sed -E 's#//go:build##; s#[&|()!]# #g' \
    | tr ' ' '\n' \
    | grep -vE '^(live_test|all)?$' || true
}

# Read groups from the first matching live-test file for a stem, or "" if none.
groups_from_stem() {
  local stem="$1" body cand
  body="$stem"; body="${body#resource_}"; body="${body#data_source_}"
  for cand in \
      "$PROVIDER_DIR/${stem}_live_test.go" \
      "$PROVIDER_DIR/resource_${body}_live_test.go" \
      "$PROVIDER_DIR/data_source_${body}_live_test.go" \
      "$PROVIDER_DIR/resource_${body%s}_live_test.go" \
      "$PROVIDER_DIR/data_source_${body%s}_live_test.go"; do
    if [ -f "$cand" ]; then
      grep -m1 '//go:build' "$cand" | extract_groups | tr '\n' ' '
      return 0
    fi
  done
  echo ""
}

# Domain keyword fallback (ordered, first match wins). Only consulted when no
# live-test file matches the stem. Keep specific patterns before generic ones.
groups_from_keyword() {
  case "$1" in
    *flink*|*fink*)                                                echo flink ;;
    *schema*|*subject*|*exporter*|*kek*|*dek*|*registry*)          echo schema_registry ;;
    *private_link*|*network*|*peering*|*gateway*|*dns*|*access_point*|*endpoint*|*transit*|*egress*|*ip_filter*|*ip_group*) echo networking ;;
    *connector*|*connect_*|*_connect*|*plugin*)                    echo connect ;;
    *kafka*|*_acl*|*mirror*|*cluster_link*|*client_quota*|*consumer_group*) echo kafka ;;
    *business_metadata*|*catalog*|*_tag*|*entity*)                 echo data_catalog ;;
    *role_binding*|*identity*|*group_mapping*|*invitation*)        echo rbac ;;
    *tableflow*)                                                   echo tableflow ;;
    *provider_integration*|*environment*|*organization*|*service_account*|*api_key*|*byok*) echo core ;;
    *)                                                             echo "" ;;
  esac
}

# --- changed files ----------------------------------------------------------
if [ -n "${CHANGED_FILES_OVERRIDE:-}" ]; then
  CHANGED=$(printf '%s\n' $CHANGED_FILES_OVERRIDE)
else
  git fetch -q origin "${BASE_REF#origin/}" 2>/dev/null || true
  if MB=$(git merge-base HEAD "$BASE_REF" 2>/dev/null); then
    CHANGED=$(git diff --name-only "$MB" HEAD || true)
  else
    log "no merge-base with $BASE_REF (shallow clone?); diffing against its tip"
    CHANGED=$(git diff --name-only "$BASE_REF" HEAD || true)
  fi
fi

if [ -z "${CHANGED// }" ]; then
  log "no changed files vs $BASE_REF"; echo "NONE"; exit 0
fi
log "changed files:"; printf '%s\n' "$CHANGED" | sed 's/^/  /' >&2

# --- walk the diff ----------------------------------------------------------
RESULT=""; BLAST=0; TOUCHED_PROVIDER=0
add_group() {
  local g
  for g in $1; do
    case ",$RESULT," in *",$g,"*) ;; *) RESULT="${RESULT:+$RESULT,}$g";; esac
  done
}

while IFS= read -r f; do
  [ -z "$f" ] && continue
  case "$f" in
    go.mod|go.sum) log "$f -> dependency bump -> blast radius"; BLAST=1; continue;;
    "$PROVIDER_DIR"/*.go) TOUCHED_PROVIDER=1;;
    *) continue;;   # docs/, examples/, ci config, etc. don't gate live tests
  esac

  stem=$(basename "$f"); stem=${stem%_live_test.go}; stem=${stem%_test.go}; stem=${stem%.go}

  g=$(groups_from_stem "$stem")
  if [ -n "${g// }" ]; then log "$f -> stem match -> groups:$g"; add_group "$g"; continue; fi

  case "$stem" in
    resource_*|data_source_*)
      g=$(groups_from_keyword "$stem")
      if [ -n "$g" ]; then log "$f -> keyword -> groups: $g"; add_group "$g"
      else log "$f -> resource/data-source with no mapping -> blast radius"; BLAST=1; fi ;;
    *)
      log "$f -> shared provider helper -> blast radius"; BLAST=1 ;;
  esac
done < <(printf '%s\n' "$CHANGED")

# --- decide -----------------------------------------------------------------
if [ "$BLAST" = "1" ]; then echo "$BLAST_RADIUS_GROUPS"; exit 0; fi
if [ -n "$RESULT" ]; then echo "$RESULT"; exit 0; fi
if [ "$TOUCHED_PROVIDER" = "1" ]; then echo "$BLAST_RADIUS_GROUPS"; else echo "NONE"; fi
