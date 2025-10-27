# Togettyc


<div style="text-align:center">
    <img src="img/Togettyc.png" alt="drawing" width="200"/>
</div>


> Cross platform reimplementation of [ttyrec](http://0xcc.net/ttyrec/) written in golang, based on [maze.io/x/ttyrec](maze.io/x/ttyrec) .

It supports:
- `rec`: Record a session
- `play`: Replay in real time the recorded session
- `time`: Print the time of the recorded sessions
- `parse`: Parse the session to display recorded session

```shell
$ togettyc -h
Usage: togettyc <command> [flags]

Cross-platform reimplementation of ttyrec

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  print [<record-file>] [flags]
    Render the record

  rec [<exe> [<args> ...]] [flags]
    Record

  play [<record-file>] [flags]
    Play the record

Run "togettyc <command> --help" for more information on a command.
```

## Rec: start a new record (equivalent to ttyrec)
```shell
$ togettyc rec -h
Usage: togettyc rec [<exe> [<args> ...]] [flags]

Record

Arguments:
  [<exe>]         Command to execute
  [<args> ...]    arguments

Flags:
  -h, --help             Show context-sensitive help.

  -a, --append           Append to the existing file
  -Z, --compress         Compress the result (zstd)
  -f, --output=STRING    Output file name
  -S, --shell=STRING     Shell to use, using current one by default
```

> `togettyc` can be used instead of `togettyc rec`

```shell
$ togettyc print
Usage: togettyc print [<record-file>] [flags]

Render the record

Arguments:
  [<record-file>]    Record file to print

Flags:
  -h, --help                     Show context-sensitive help.

  -d, --date                     Show date
      --no-color                 Disable colors
  -H, --html                     Display result in HTML
  -S, --start-date=LOCAL-TIME    Show results after the provided date (format:"YYYY-MM-DD hh:mm:ss")
  -E, --end-date=LOCAL-TIME      Show results before the provided date (format:"YYYY-MM-DD hh:mm:ss")
  -T, --tmux                     Clean the output with tmux. It should reduce the noise provoked by garbage terminal manipulation
```



## Play: replay a record (equivalent to play)
```
$ togettyc play
Usage: togettyc play [<record-file>] [flags]

Play the record

Arguments:
  [<record-file>]    Record file to replay

Flags:
  -h, --help         Show context-sensitive help.

  -s, --speed=1.0    Modify the speed
  -n, --no-wait      No wait mode
```

## Time: print the time of the record

```shell
Usage: togettyc time [<record-file> ...] [flags]

Time to record

Arguments:
  [<record-file> ...]    Record file to replay

Flags:
  -h, --help              Show context-sensitive help.

  -h, --human-readable    Print human readable time
```

## Print: Render the record

```shell
Usage: togettyc print [<record-file>] [flags]

Render the record

Arguments:
  [<record-file>]    Record file to print

Flags:
  -h, --help                     Show context-sensitive help.

  -d, --date                     Show date
      --no-color                 Disable colors
  -H, --html                     Display result in HTML
  -S, --start-date=LOCAL-TIME    Show results after the provided date (format:"YYYY-MM-DD hh:mm:ss")
  -E, --end-date=LOCAL-TIME      Show results before the provided date (format:"YYYY-MM-DD hh:mm:ss")
  -T, --tmux                     Clean the output with tmux. It should reduce the noise provoked by garbage terminal manipulation
```