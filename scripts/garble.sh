#!/bin/sh

if [[ "$NO_LITERALS" == 1 ]]; then
  garble -seed="$CLIENT__COMPILE_SEED" -tiny "$@"
else
  garble -literals -seed="$CLIENT__COMPILE_SEED"  -tiny "$@"
fi