#!/bin/sh
set -e
rm -rf manpages
mkdir manpages
go run ./cmd/wheresmyprompt/ man | gzip -c -9 >manpages/wheresmyprompt.1.gz
