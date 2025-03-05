JSON Lines Viewer
=================

Converts JSON lines from input into key-value output.

Keys are sorted by predefined order or user-defined order 
which makes for consistent
output. Some important keys are printed first.

Input that is not JSON is not processed and is printed as-is.

Usage
-----

This is mostly useful for viewing log files:

* `tail -f log-file.json | json-viewer`
* `./start-server 2>&1 | json-viewer`
