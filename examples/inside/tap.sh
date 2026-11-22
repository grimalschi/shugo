#!/usr/bin/env bash
# Runs the JavaScript of the project the way shugo does, and keeps a copy of
# the conversation: what the JavaScript side says to shugo (descriptor 3)
# and what it hears from it (descriptor 4). SHUGO_TAP names the folder.
set -e
# Only evaluation opens the two protocol descriptors.
if ! { : >&3 && : <&4; } 2>/dev/null; then
  exec bun "$@"
fi
folder="${SHUGO_TAP:?the folder for the copy}"
mkdir -p "$folder"
exec bun "$@" 3> >(tee -a "$folder/answers.bin" >&3) 4< <(tee -a "$folder/requests.bin" <&4)
