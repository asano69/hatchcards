## hashcards --help
```
A plain text-based spaced repetition system.

Usage: hashcards <COMMAND>

Commands:
  drill    Drill cards through a web interface
  check    Check the integrity of a collection
  stats    Print collection statistics
  orphans  Commands relating to orphan cards
  export   Export a collection
  help     Print this message or the help of the given subcommand(s)

Options:
  -h, --help     Print help
  -V, --version  Print version
```

## hashcards drill --help
```
Drill cards through a web interface

Usage: hashcards drill [OPTIONS] [DIRECTORY]

Arguments:
  [DIRECTORY]
          Path to the collection directory. By default, the current working directory is used

Options:
      --card-limit <CARD_LIMIT>
          Maximum number of cards to drill in a session. By default, all cards due today are drilled

      --new-card-limit <NEW_CARD_LIMIT>
          Maximum number of new cards to drill in a session

      --host <HOST>
          The host address to bind to. Default is 127.0.0.1

          [default: 127.0.0.1]

      --port <PORT>
          The port to use for the web server. Default is 8000

          [default: 8000]

      --from-deck <FROM_DECK>
          Only drill cards from this deck

      --open-browser <OPEN_BROWSER>
          Whether to open the browser automatically. Default is true

          [possible values: true, false]

      --answer-controls <ANSWER_CONTROLS>
          Which answer controls to show:

          Possible values:
          - full:   Show all four rating buttons (Forgot/Hard/Good/Easy)
          - binary: Show only two rating buttons (Forgot/Good)

          [default: full]

      --bury-siblings <BURY_SIBLINGS>
          Whether or not to bury siblings. Default is true

          [possible values: true, false]

  -h, --help
          Print help (see a summary with '-h')
```

## hashcards check --help
```
Check the integrity of a collection

Usage: hashcards check [DIRECTORY]

Arguments:
  [DIRECTORY]  Path to the collection directory. By default, the current working directory is used

Options:
  -h, --help  Print help
```

## hashcards stats --help
```
Print collection statistics

Usage: hashcards stats [OPTIONS] [DIRECTORY]

Arguments:
  [DIRECTORY]
          Path to the collection directory. By default, the current working directory is used

Options:
      --format <FORMAT>
          Which output format to use

          Possible values:
          - html: HTML output
          - json: JSON output

          [default: html]

  -h, --help
          Print help (see a summary with '-h')

```

## hashcards orphans --help
```
Commands relating to orphan cards

Usage: hashcards orphans <COMMAND>

Commands:
  list    List the hashes of all orphan cards in the collection
  delete  Remove all orphan cards from the database
  help    Print this message or the help of the given subcommand(s)

Options:
  -h, --help  Print help
```

## hashcards export --help
```
Export a collection

Usage: hashcards export [OPTIONS] [DIRECTORY]

Arguments:
  [DIRECTORY]  Path to the collection directory. By default, the current working directory is used

Options:
      --output <OUTPUT>  Optional path to the output file. By default, the output is printed to stdout
  -h, --help             Print help
```

## hashcards -V
```
hashcards 0.3.0
```
