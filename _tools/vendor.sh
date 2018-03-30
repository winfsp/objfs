#!/bin/bash

Hosts="(github\.com|bitbucket\.org|gitlab\.com)"
Root=$(cd "$0/../.." | pwd)

usage() {
    echo "usage: vendor.sh {list|add} dir..." 1>&2
    exit 2
}

subadd() {
    (cd "$Root" && git submodule add "https://$1.git" "vendor/$1")
}

case "$1" in
add)    Command="subadd"; shift;;
list)   Command="echo"; shift;;
*)      usage;;
esac
[[ $# -gt 0 ]] || usage

for a in "$@"; do
    (
        cd "$a"
        Package=$(go list | sed -E -n -e 's@^([^/]*/[^/]*/[^/]*).*$@\1@p')
        go list -f '{{join .Deps "\n"}}' |
            sed -E -n \
                -e '\'"@^$Package@d" \
                -e '\'"@^$Hosts@"'s@^([^/]*/[^/]*/[^/]*).*$@\1@p'
    )
done | sort | uniq |
while read Package; do
    if [[ ! -e "$Root/vendor/$Package" ]]; then
        $Command $Package
    fi
done
