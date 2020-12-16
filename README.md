```
$ ./estk -h
Usage: estk --url=STRING <command>

Flags:
  -h, --help          Show context-sensitive help.
      --url=STRING    Base Kibana/ES url
  -q, --quiet         Quiet mode

Commands:
  list --url=STRING
    List indices

  dump --url=STRING --index=STRING
    Dump indices

Run "estk <command> --help" for more information on a command.

$ ./estk dump -h
Usage: estk dump --url=STRING --index=STRING

Dump indices

Flags:
  -h, --help                   Show context-sensitive help.
      --url=STRING             Base Kibana/ES url
  -q, --quiet                  Quiet mode

      --index=STRING           Index filter
      --size="100"             Bulk size
  -o, --output-file=STRING     Output file
      --query-string=STRING    Query string to filter results

```