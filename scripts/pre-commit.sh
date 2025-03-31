#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

"$SCRIPT_DIR/pre-commit/gen_build_config.sh"
"$SCRIPT_DIR/pre-commit/gen_config.sh"
"$SCRIPT_DIR/pre-commit/update_readme.sh"
