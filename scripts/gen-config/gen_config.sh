#!/bin/bash

function gen(){
  local target
  target="$1"
  echo Generating "$target" configuration file...
  mkdir -p ./config
  go run ./"$target" --generate-config > ./config/"$target"_config.yaml
  echo "Done"
}


gen client
gen agent
gen server