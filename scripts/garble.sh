#!/bin/sh
FLAG=()
if [[ "$TINY" == "true" ]];then
  FLAG+=("-tiny")
fi

if [[ "$LITERALS" == "true" ]] && [[ "$NO_LITERALS" != 1 ]]; then
  FLAG+=("-literals")
fi

echo garble -seed="$CLIENT__COMPILE_SEED" ${FLAG[@]+"${FLAG[@]}"} "$@"
garble -seed="$CLIENT__COMPILE_SEED" ${FLAG[@]+"${FLAG[@]}"} "$@"
