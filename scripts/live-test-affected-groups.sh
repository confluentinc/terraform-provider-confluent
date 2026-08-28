#!/usr/bin/env bash
#
# scripts/live-test-affected-groups.sh
#
# Works out which live tests are relevant to this branch's diff, so a PR can run
# just those instead of a whole build-tag group (or the full nightly suite).
#
# It prints TWO lines to stdout:
#   GROUPS=<NONE | all | comma-list>   the build-tag groups to compile
#   RUN=<regex | empty>                a -run regex naming the exact tests, or
#                                      empty to run every Live test in GROUPS
#
#   GROUPS=NONE   -> no provider code changed; caller should skip live tests
#   GROUPS=all    -> a shared / unrecognized file changed; run everything
#   RUN set       -> every changed file mapped to concrete live tests (precise)
#   RUN empty     -> at least one change could only be pinned to a whole group
#
# For each changed internal/provider/*.go file it finds the matching
# *_live_test.go file(s) -- the file's own stem, its resource<->data_source
# sibling, and a singular form -- and:
#   * reads their //go:build groups (tolerating files that have no build tag);
#   * asks scripts/livetestfuncs for the exact Test*Live / *DriftDetection funcs
#     they declare (AST, not a text scan, so odd formatting / missing build tags
#     are handled) and runs precisely those.
# Files with no matching live test fall back to a domain keyword group (whole
# group, no -run); shared or unrecognized files fall back to "all". It never
# *under*-runs; the nightly full run remains the backstop.
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
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { echo "[affected-groups] $*" >&2; }

# "//go:build live_test && (all || kafka || core)" -> lines: kafka, core
extract_groups() {
  sed -E 's#//go:build##; s#[&|()!]# #g' \
    | tr ' ' '\n' \
    | grep -vE '^(live_test|all)?$' || true
}

# Existing *_live_test.go file(s) for a stem (may be zero, one, or two paths).
matched_live_files() {
  local stem="$1" body cand out=""
  body="$stem"
  body="${body#resource_}"
  body="${body#data_source_}"
  for cand in \
      "$PROVIDER_DIR/${stem}_live_test.go" \
      "$PROVIDER_DIR/resource_${body}_live_test.go" \
      "$PROVIDER_DIR/data_source_${body}_live_test.go" \
      "$PROVIDER_DIR/resource_${body%s}_live_test.go" \
      "$PROVIDER_DIR/data_source_${body%s}_live_test.go"; do
    if [ -f "$cand" ] && [[ " $out " != *" $cand "* ]]; then
      out="${out:+$out }$cand"
    fi
  done
  echo "$out"
}

# Build-tag groups of a single file, or "" if it has no //go:build line.
# The `|| true` is load-bearing: some live-test files (e.g.
# flink_compute_pool_config) have no build tag, and without it `set -o pipefail`
# would abort the whole script.
groups_of_file() {
  local tag
  tag=$(grep -m1 '//go:build' "$1" 2>/dev/null || true)
  [ -n "$tag" ] || { echo ""; return; }
  printf '%s\n' "$tag" | extract_groups | tr '\n' ' '
}

# Domain keyword fallback (ordered, first match wins). Consulted only when no
# live-test file matches, or when a matched file carries no build tag.
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

emit() { echo "GROUPS=$1"; echo "RUN=${2:-}"; }

# --- changed files ----------------------------------------------------------
if [ -n "${CHANGED_FILES_OVERRIDE:-}" ]; then
  # Quote the expansion (no word-splitting/globbing) and normalize spaces to
  # newlines, so an override like "a b" or "pkg/*" is taken literally.
  CHANGED=$(printf '%s' "$CHANGED_FILES_OVERRIDE" | tr ' ' '\n')
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
  log "no changed files vs $BASE_REF"; emit "NONE"; exit 0
fi
log "changed files:"; printf '%s\n' "$CHANGED" | sed 's/^/  /' >&2

# --- walk the diff ----------------------------------------------------------
# NB: do NOT name the group accumulator GROUPS -- that is a reserved bash
# variable (the caller's group IDs) and assignments to it are ignored.
GRPS=""
FILES=""
BLAST=0
GROUP_LEVEL=0
TOUCHED_PROVIDER=0

add_grp() {
  local g
  for g in $1; do
    if [[ ",$GRPS," != *",$g,"* ]]; then GRPS="${GRPS:+$GRPS,}$g"; fi
  done
}
add_file() {
  if [[ " $FILES " != *" $1 "* ]]; then FILES="${FILES:+$FILES }$1"; fi
}

while IFS= read -r f; do
  [ -z "$f" ] && continue
  case "$f" in
    go.mod|go.sum) log "$f -> dependency bump -> blast radius"; BLAST=1; continue;;
    "$PROVIDER_DIR"/*.go) TOUCHED_PROVIDER=1;;
    *) continue;;   # docs/, examples/, ci config, etc. don't gate live tests
  esac

  stem=$(basename "$f"); stem=${stem%_live_test.go}; stem=${stem%_test.go}; stem=${stem%.go}
  files=$(matched_live_files "$stem")

  if [ -n "${files// }" ]; then
    for lf in $files; do
      add_file "$lf"
      fg=$(groups_of_file "$lf")
      if [ -n "${fg// }" ]; then
        add_grp "$fg"
      else
        kg=$(groups_from_keyword "$stem")
        if [ -n "$kg" ]; then add_grp "$kg"; else log "$f -> matched untagged file, no keyword -> blast"; BLAST=1; fi
      fi
    done
    log "$f -> live files:$files"
  else
    case "$stem" in
      resource_*|data_source_*)
        kg=$(groups_from_keyword "$stem")
        if [ -n "$kg" ]; then log "$f -> no live file; keyword -> group $kg (whole group)"; add_grp "$kg"; GROUP_LEVEL=1
        else log "$f -> resource/data-source with no mapping -> blast"; BLAST=1; fi ;;
      *) log "$f -> shared provider helper -> blast"; BLAST=1 ;;
    esac
  fi
done < <(printf '%s\n' "$CHANGED")

# --- decide -----------------------------------------------------------------
if [ "$BLAST" = "1" ]; then emit "$BLAST_RADIUS_GROUPS"; exit 0; fi
if [ -z "$GRPS" ]; then
  if [ "$TOUCHED_PROVIDER" = "1" ]; then emit "$BLAST_RADIUS_GROUPS"; else emit "NONE"; fi
  exit 0
fi

# Concrete groups. Go precise only if EVERY change pinned to a live file.
if [ "$GROUP_LEVEL" = "1" ] || [ -z "${FILES// }" ]; then
  log "at least one change is group-level; running whole group(s): $GRPS"
  emit "$GRPS"; exit 0
fi

NAMES=$(cd "$SCRIPT_DIR/.." && go run ./scripts/livetestfuncs $FILES 2>/dev/null | sort -u || true)
if [ -z "${NAMES// }" ]; then
  log "could not extract test names (falling back to whole group(s): $GRPS)"
  emit "$GRPS"; exit 0
fi
RUN="^($(printf '%s' "$NAMES" | paste -sd'|' -))$"
log "precise run for groups [$GRPS]: $RUN"
emit "$GRPS" "$RUN"
