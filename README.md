# Mock Exporter

This project can provide multiple listening ports by specifying a sample input file to provide services similar to `/metrics`

## Usage

```bash
$ ./mock_exporter -h
usage: mock-exporter --mock=MOCK [<flags>]


Flags:
  -h, --[no-]help          Show context-sensitive help (also try --help-long and
                           --help-man).
  -p, --path="/metrics"    Path under which to expose metrics.
  -m, --mock=MOCK          Sample prom file (.prom) that requires mocking
      --web.port=10000     The starting value of the port
      --web.length=50      The length of the port range (starting from the
                           starting value. If any port is occupied, it will be
                           skipped.)
      --log.level=info     Only log messages with the given severity or above.
                           One of: [debug, info, warn, error]
      --log.format=logfmt  Output format of log messages. One of: [logfmt, json]
      --[no-]version       Show application version.
```

## Example

```bash
$ ./mock_exporter -m ./node-exporter.prom
```

## Build

```bash
$ go build -o mock_exporter
```
