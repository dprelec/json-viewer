JSON Lines Viewer
=================

Converts JSON lines from input into key-value output.

Keys found in the top object are sorted by predefined order.
User-defined order is also supported with, for example, `--order=k1,k2,k3`.
Ordering output by predefined keys makes for consistent output. 

Input that is not JSON is not processed and is printed as-is.

Key names, and some magical values will get their own color.

Usage
-----

This is mostly useful for viewing log files:

* `tail -f log-file.json | json-viewer`
* `./start-server 2>&1 | json-viewer`

Example
-------

Document like:
```json
{"time":"05.03.2025. 17:01:19 CET", "level":"info","status":404}
```

will be printed like:

```
level=info
time=05.03.2025. 17:01:19 CET
status=404
```
