#!/bin/sh
FLAG=()
if [[ "$TINY" == "true" ]];then
  FLAG+=("-tiny")
fi

if [[ "$LITERALS" == "true" ]] && [[ "$NO_LITERALS" != 1 ]]; then
  FLAG+=("-literals")
fi

echo garble -debugdir=out -seed="$CLIENT__COMPILE_SEED" ${FLAG[@]+"${FLAG[@]}"} "$@"
garble -debugdir=out -seed="$CLIENT__COMPILE_SEED" ${FLAG[@]+"${FLAG[@]}"} "$@"
