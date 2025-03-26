#!/bin/bash

ENV_FILE="env.test"
GORELEASER_ENV="goreleaser.env"
GORELEASER_FLAGS=""

function GenConfig(){
  source_file="$1"

  var_file="${2}"

  module="$3"

  while read -r var;do
    if grep -q  '^$' <<< "$var" ;then
      echo >> "$ENV_FILE"
      echo >> "$GORELEASER_ENV"
      echo >> "$GORELEASER_FLAGS"
      continue
    fi
    if ! grep -q "$var.*=" $var_file;then
      echo "Variable $var not defined in $var_file"
    fi
    if ! grep -qE  "\"$var\": +$var," "$var_file";then
      echo "Variable $var not defined in default vars in $var_file"
    fi


    ID="${source_file%%/*}"
    PREFIX="$(echo "$ID" | tr '[:lower:]' '[:upper:]')"
    ENV_VAR="${PREFIX}_$(echo "$var" | tr '[:lower:]' '[:upper:]')"

    echo "$ENV_VAR=" >> "$ENV_FILE"
    echo "  - $ENV_VAR={{ .Env.$ENV_VAR }}" >> "$GORELEASER_ENV"

    package="$(head -n 1 "$source_file" | awk '{print $2}')"


    echo "      - '{{ if (index .Env \"$ENV_VAR\") }} -X $module.$var={{ .Env.$ENV_VAR }}{{end}}'" >> "$GORELEASER_FLAGS"
  done < <(cat "$source_file" | rg -A 1 'default:"\$\{(.*)\}"' -oa -r '$1' | sed '$d')
}

: > "$ENV_FILE"
: > "$GORELEASER_ENV"

echo "" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"
echo "##### SERVER CONFIGURATION #####" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"

GORELEASER_FLAGS="goreleaser_flag_server.yaml"
: > "$GORELEASER_FLAGS"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_ENV"
echo >> "$GORELEASER_ENV"
GenConfig "server/config/config.go" "server/config/config.go" "Goauld/server/config"



echo "" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"
echo "##### AGENT CONFIGURATION  #####" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"


GORELEASER_FLAGS="goreleaser_flag_agent.yaml"
: > "$GORELEASER_FLAGS"
echo >> "$GORELEASER_ENV"
echo >> "$GORELEASER_ENV"
GenConfig "agent/config/config.go" "agent/config/config.go" "Goauld/agent/config"


echo "" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"
echo "##### CLIENT CONFIGURATION #####" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"

GORELEASER_FLAGS="goreleaser_flag_client.yaml"
: > "$GORELEASER_FLAGS"
GenConfig "client/config.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"

GenConfig "client/Ssh.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"

GenConfig "client/Socks.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"

GenConfig "client/Scp.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"

GenConfig "client/Pass.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"

GenConfig "client/compiler/compiler.go" "client/config.go" "main"
echo >> "$ENV_FILE"
echo >> "$GORELEASER_FLAGS"
