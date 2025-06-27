#!/bin/bash
set -ueo pipefail
ENV_FILE="$(mktemp)"
GORELEASER_ENV="$(mktemp)"
GORELEASER_FLAGS=""
GORELEASER_FLAGS_SERVER="$(mktemp)"
GORELEASER_FLAGS_AGENT="$(mktemp)"
GORELEASER_FLAGS_CLIENT="$(mktemp)"

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

    help="$(rg -o "default:\"\\$\{$var}\".*help:\"(.*?)\"" -r '$1' "$source_file"  )"
    if [[ "$help" == "-" ]];then
      continue
    fi

    echo "# $help" >> "$ENV_FILE"
    echo "$ENV_VAR=" >> "$ENV_FILE"
    echo "  - $ENV_VAR={{ .Env.$ENV_VAR }}" >> "$GORELEASER_ENV"


    echo "      - '{{ if (index .Env \"$ENV_VAR\") }} -X $module.$var={{ .Env.$ENV_VAR }}{{end}}'" >> "$GORELEASER_FLAGS"
  done < <(cat "$source_file" | rg -A 1 'default:"(\$\{(.*)\})?"' -oa -r '$2' | sed '$d')
}

: > "$ENV_FILE"
: > "$GORELEASER_ENV"

echo "" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"
echo "##### SERVER CONFIGURATION #####" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"

GORELEASER_FLAGS="$GORELEASER_FLAGS_SERVER"
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


GORELEASER_FLAGS="$GORELEASER_FLAGS_AGENT"
: > "$GORELEASER_FLAGS"
echo >> "$GORELEASER_ENV"
echo >> "$GORELEASER_ENV"
GenConfig "agent/config/config.go" "agent/config/config.go" "Goauld/agent/config"


echo "" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"
echo "##### CLIENT CONFIGURATION #####" >> "$ENV_FILE"
echo "################################" >> "$ENV_FILE"

GORELEASER_FLAGS="$GORELEASER_FLAGS_CLIENT"
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


function UpdateContent(){
  # Check if correct number of arguments are provided
  if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <marker> <file> <new_content>"
    exit 1
  fi

  # Parameters passed to the script
  MARKER="$1"
  FILE="$2"
  NEW_CONTENT="$(cat "$3")"
  NEW_CONTENT="$(echo "$NEW_CONTENT" | sed 's/$/\\n/g' | tr -d '\n')"

  # Temporary file
  TEMP_FILE=$(mktemp)

  # Replace content between markers
  awk -v marker="$MARKER" -v new_content="$NEW_CONTENT" '
  BEGIN {
      in_block = 0
      print_content = 1
  }
  {
      if ($0 ~ "# BEGIN Dynamic " marker) {
          print
          print new_content
          in_block = 1
          print_content = 0
          next
      }

      if ($0 ~ "# END Dynamic " marker) {
          in_block = 0
          print_content = 1
      }

      if (!in_block && print_content) {
          print
      }
  }
  ' "$FILE" > "$TEMP_FILE" && mv "$TEMP_FILE" "$FILE"

  echo "Content replaced successfully in $FILE"
}

UpdateContent "ENV" ./.goreleaser.yaml "$GORELEASER_ENV"
UpdateContent "SERVER" ./.goreleaser.yaml "$GORELEASER_FLAGS_SERVER"
UpdateContent "CLIENT" ./.goreleaser.yaml "$GORELEASER_FLAGS_CLIENT"
UpdateContent "AGENT" ./.goreleaser.yaml "$GORELEASER_FLAGS_AGENT"

UpdateContent "ENV" ./.env.build.tmpl "$ENV_FILE"



rm -rf "$ENV_FILE"
rm -rf "$GORELEASER_ENV"
rm -rf "$GORELEASER_FLAGS_SERVER"
rm -rf "$GORELEASER_FLAGS_AGENT"
rm -rf "$GORELEASER_FLAGS_CLIENT"