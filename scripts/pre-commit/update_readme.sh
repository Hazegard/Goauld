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
UpdateBlocCode Readme.md 'Agent (`goauld`)' <(echo '$ goauld --help'; COLUMNS=10000 go run ./agent --help | cat | sed -n '/^Usage:/,$p')

# shellcheck disable=SC2016
echo 'Client (`tealc`)'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Client (`tealc`)' <(echo '$ tealc --help'; COLUMNS=10000 go run ./client --help | cat| sed -n '/^Usage:/,$p')

echo 'TUI'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'TUI' <(echo '$ tealc tui --help'; COLUMNS=10000 go run ./client tui --help | cat| sed -n '/^Usage:/,$p')

echo 'SSH (exec)'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'SSH (exec)' <(echo '$ tealc ssh --help'; COLUMNS=10000 go run ./client ssh --help | cat| sed -n '/^Usage:/,$p')

echo 'SCP'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'SCP' <(echo '$ tealc scp --help'; COLUMNS=10000 go run ./client scp --help | cat| sed -n '/^Usage:/,$p')

echo 'Compile'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Compile' <(echo '$ tealc compile --help'; COLUMNS=10000 go run ./client compile --help | cat| sed -n '/^Usage:/,$p' | sed 's/Usage: tealc \[flags\]/Usage: tealc compile [flags]/g')

echo 'Server'
# shellcheck disable=SC2016
UpdateBlocCode Readme.md 'Server' <(echo '$ goauld_server --help'; COLUMNS=10000 go run ./server --help | cat | sed -n '/^Usage:/,$p')