#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

me=$(basename "$0")
bin=$(cd "$(dirname "$0")" && /bin/pwd)


err()	{ echo >&2 "$*"; }
log()	{ err  "$me: $*"; }
vlog()	{ $verbose && log "$*"; }
fail()	{ log "$*"; exit 1; }
rawecho() { echo -E "x$*" | sed 's/^x//'; }
prefix()  { sed "s/^/$me: /"; }

sfc_usage() {
  err
  err "usage:"
  err "  $me [options]"
  err
  err "options:"
  err "  -p,--profile=<profile> -- comma sep list of config profile(s)"
  err "  -h --help              -- this help message"
  err
  exit 1
}

onload_set() {
  if [ -z "$2" ]; then
    # it's not in onload_set x y form - try onload_set x=y
    VAR=$(echo "$1" | cut -f1 -d=)
    VAL=$(echo "$1" | cut -f2 -d=)
  else
    VAR="$1"
    VAL="$2"
  fi
  echo "  $VAR: \"$VAL\""
}

onload_import() {
  # Called below to import a profile.  May also be called by config scripts
  # and profile scripts to import another profile.
  local profile="$1"
  local pf1="$HOME/.openonload/profiles/$profile.opf"
  local pf3="$profile"
  # *.opf-fragment files are searched only for secondary imports, i.e. use of
  # onload_import within another profile. This is to prevent fragments being
  # accidentally used as full-fledged profiles by themselves
  local pf_frag1="$HOME/.openonload/profiles/$profile.opf-fragment"
  local pf="$pf1"
  $toplevelimport || [ -f "$pf" ] || pf="$pf_frag1"
  for dir in $profile_d; do
    local pf2="$dir/$profile.opf"
    [ -f "$pf" ] || pf="$pf2"
    local pf_frag2="$dir/$profile.opf-fragment"
    $toplevelimport || [ -f "$pf" ] || pf="$pf_frag2"
  done
  [ -f "$pf" ] || pf="$pf3"
  if ! [ -f "$pf" ]; then
    log "ERROR: Cannot find profile '$profile'"
    log "I looked in these places:"
    log "  $pf1"
    $toplevelimport || log "  $pf_frag1"
    for dir in $profile_d; do
      local pf2="$dir/$profile.opf"
      log "  $pf2"
      local pf_frag2="$dir/$profile.opf-fragment"
      $toplevelimport || log "  $pf_frag2"
    done
    log "  $pf3"
    exit 3
  fi
  toplevelimport=false
  vlog "onload_import: $profile ($pf)"
  # Source profile, with $@ representing the application and its options
  shift
  # shellcheck disable=SC1090
  . "$pf"
  vlog "onload_import-done: $profile"
}

######################################################################
# main()

if [ -x "$bin/mmaketool" ]; then
  profile_d="$bin/onload_profiles"
else
  profile_d=/usr/libexec/onload/profiles
fi

profiles=
verbose=false

while [ $# -gt 0 ]; do
  case "$1" in
    --profile=*)
	onload_args="$onload_args $1"
	profile="${1#--profile=}"
	[ -n "$profile" ] || sfc_usage
	profiles="$profiles ${profile//,/ }"
	;;
    --profile|-p)
	onload_args="$onload_args $1 $2"
	shift
	profile="$1"
	[ -n "$profile" ] || sfc_usage
	profiles="$profiles ${profile//,/ }"
	;;
    -*)	sfc_usage
	;;
  esac
  shift
done

[ $# = 0 ] && [ -z "$profiles" ] && sfc_usage

for profile in $profiles; do
  profile_d="$profile_d $(dirname "$profile")"
done

for profile in $profiles; do
  echo "apiVersion: v1"
  echo "kind: ConfigMap"
  echo "metadata:"
  echo "  name: onload-$(basename "$profile" .opf)-profile"
  echo "data:"
  onload_import "$profile" "$@"
  echo "---"
done
