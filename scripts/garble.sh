#!/bin/sh

if [[ "$NO_LITERALS" == 1 ]]; then
  garble -seed="$CLIENT__COMPILE_SEED" "$@"
else
  garble -literals -seed="$CLIENT__COMPILE_SEED" "$@"
fi