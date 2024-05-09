JSON Viewer
===========

Converts JSON input into key-value output.
Keys are sorted by predefined or user-defined order which makes for consistent
output. Important keys are printed first.

Input that is not JSON is not processed and is printed as-is.

Usage
-----

This is mostly useful for viewing log files:

* `tail -f log-file.json | json-viewer`
* `./start-server 2>&1 | json-viewer`
