#!/bin/sh
TINY_FLAG=()
if [[ "$TINY" == "true" ]];then
  TINY_FLAG=(-tiny)
fi

if [[ "$LITERALS" == "true" ]]; then
  garble -literals -seed="$CLIENT__COMPILE_SEED" ${TINY_FLAG[@]+"${TINY_FLAG[@]}"} "$@"
else
  garble -seed="$CLIENT__COMPILE_SEED" ${TINY_FLAG[@]+"${TINY_FLAG[@]}"} "$@"
fi