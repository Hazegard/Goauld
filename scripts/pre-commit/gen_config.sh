#!/bin/bash

function gen(){
  local target
  target="$1"
  echo Generating "$target" configuration file...
  mkdir -p ./config
  # Generate config, remove nul byte, remove empty line in a command and remove the first line if it is empty
  go run ./"$target" --generate-config | tr -d '\000' | awk 'NF==0 && prev~/^#/ {next} {if(!(prev~/^#/ && NF==0 && $0!~/^#/)) print prev} {prev=$0} END{print prev}'  | awk 'NR==1 && NF==0 {next} {print}' > ./config/"$target"_config.yaml
  echo "Done"
}


gen client
gen agent
gen server