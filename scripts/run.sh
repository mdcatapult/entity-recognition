#! /bin/bash
set -m

trap ctrl_c INT

function ctrl_c() {
  if [[ $(jobs -l) ]]; then
    fg
  else
    exit 0
  fi
}

bin/dictionary &
bin/regexer &
bin/recognition-api