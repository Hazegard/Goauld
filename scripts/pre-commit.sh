#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

"$SCRIPT_DIR/pre-commit/gen_build_config.sh"
"$SCRIPT_DIR/pre-commit/gen_config.sh"
# "$SCRIPT_DIR/pre-commit/update_readme.sh"


MISSING="$(
  {
    ENVVAR="$(go run ./agent -h | tr -d '\000' | rg '\$GOAULD_(.*)\)' -or '"$1"')"
    rg -Nof <(echo "$ENVVAR") ./agent/config/config_mini.go
    echo "$ENVVAR"
  } | sort | uniq -u
)"
if [[ -n "$MISSING" ]]; then
  echo "Missing prefix env in ./agent/config/config_mini.go"
  echo "$MISSING"
fi