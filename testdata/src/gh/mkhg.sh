#!/bin/sh
#
# Generate GitHub test/benchmark data.
#

set -euC

root=$(dirname "$0")
out="$root/gh.go"

IFS="
"

printf 'package gh\n\n' >| "$out"

for t in $(grep -o 'type \w\+ struct' "$root/../github.com/google/go-github/github/"*.go | cut -d' ' -f2); do
	cat <<EOF >>"$out"
// GET /$t
// $t
//
// Response 200: \$ref: github.com/google/go-github/github.$t

EOF
done
