#!/bin/bash


function UpdateBlocCode(){
  # Usage: ./replace_code_block.sh <input_file> <subtitle> <new_code_file>

  # echo $1
  # echo $2
  # cat $3
  INPUT_FILE="$1"
  SUBTITLE="$2"
  NEW_CODE_FILE="$3"
  TMP_FILE="$(mktemp)"

  # if [[ ! -f "$INPUT_FILE" || ! -f "$NEW_CODE_FILE" ]]; then
  #   echo "Error: Input file or new code file does not exist."
  #   exit 1
  # fi

  awk -v subtitle="$SUBTITLE" -v code_file="$NEW_CODE_FILE" '
  BEGIN {
    found_subtitle = 0;
    in_code_block = 0;
    replaced = 0;
  }
  {
    # Match heading of any level
    if ($0 ~ /^#{1,6} /) {
      heading = substr($0, index($0, " ") + 1)
      if (heading == subtitle) {
        found_subtitle = 1;
      } else {
        found_subtitle = 0;
      }
      print;
      next;
    }

    # Replace the first code block after the matching subtitle
    if (found_subtitle && $0 ~ /^```/) {
      if (!in_code_block) {
        in_code_block = 1;
        print $0; # Print the opening ```

        # Inject new code
        while ((getline line < code_file) > 0) {
          print line;
        }
        close(code_file);

        # Skip old block until closing ```
        while ((getline line) > 0) {
          if (line ~ /^```/) {
            print line;
            break;
          }
        }

        found_subtitle = 0;
        replaced = 1;
        next;
      }
    }

    print;
  }
  END {
    if (!replaced) {
      print "Warning: Subtitle not found or code block not replaced." > "/dev/stderr";
    }
  }
  ' "$INPUT_FILE" > "$TMP_FILE" && mv "$TMP_FILE" "$INPUT_FILE"
}

# shellcheck disable=SC2016
echo 'Agent (`goauld`)'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Agent (`goauld`)' <(echo '$ goauld --help'; COLUMNS=10000 go run ./agent --help | cat | sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g"  | tr -d '\000')

# shellcheck disable=SC2016
echo 'Client (`tealc`)'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Client (`tealc`)' <(echo '$ tealc --help'; COLUMNS=10000 go run ./client --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'TUI'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'TUI' <(echo '$ tealc tui --help'; COLUMNS=10000 go run ./client tui --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'SSH (exec)'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'SSH (exec)' <(echo '$ tealc ssh --help'; COLUMNS=10000 go run ./client ssh --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'SCP'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'SCP' <(echo '$ tealc scp --help'; COLUMNS=10000 go run ./client scp --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'SOCKS'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'SOCKS' <(echo '$ tealc socks --help'; COLUMNS=10000 go run ./client socks --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'RSYNC'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'RSYNC' <(echo '$ tealc rsync --help'; COLUMNS=10000 go run ./client rsync --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'JUMP'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'JUMP' <(echo '$ tealc jump --help'; COLUMNS=10000 go run ./client jump --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'CLIP GET'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'CLIP GET' <(echo '$ tealc clip get --help'; COLUMNS=10000 go run ./client clip get --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'CLIP SET'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'CLIP SET' <(echo '$ tealc clip set --help'; COLUMNS=10000 go run ./client clip set --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'KILL'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'KILL' <(echo '$ tealc kill --help'; COLUMNS=10000 go run ./client kill --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')
echo 'DELETE'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'DELETE' <(echo '$ tealc delete --help'; COLUMNS=10000 go run ./client delete --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'LIST'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'LIST' <(echo '$ tealc list --help'; COLUMNS=10000 go run ./client list --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'WIREGUARD GENERATE'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'WIREGUARD GENERATE' <(echo '$ tealc wireguard generate --help'; COLUMNS=10000 go run ./client wireguard generate --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'WIREGUARD START'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'WIREGUARD START' <(echo '$ tealc wireguard start --help'; COLUMNS=10000 go run ./client wireguard start --help | cat| sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'Compile'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Compile' <(echo '$ tealc compile --help'; COLUMNS=10000 go run ./client compile -O darwin --help | cat| sed -n '/^Usage:/,$p' |grep -v "@" | sed 's/Usage: tealc \[flags\]/Usage: tealc compile [flags]/g' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')

echo 'Server'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Server' <(echo '$ goauld_server --help'; COLUMNS=10000 go run ./server --help | cat | sed -n '/^Usage:/,$p' | sed "s/Goa'uld/Goauld/g" | tr -d '\000')