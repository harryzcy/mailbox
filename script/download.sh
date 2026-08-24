#!/bin/bash

set -euo pipefail

tag_name=$(
    curl -s https://api.github.com/repos/harryzcy/mailbox/releases/latest |
        grep "tag_name" |
        cut -d : -f 2,3 |
        tr -d "\",[:space:]"
)

# Records which release bin/ holds, so a re-run downloads nothing when it is
# already current. Removed by make clean, and absent after a local build.
stamp="bin/.release"

if [ -f "${stamp}" ] && [ "$(cat "${stamp}")" == "${tag_name}" ]; then
    echo "bin/ already holds ${tag_name}"
    exit 0
fi

url="https://github.com/harryzcy/mailbox/releases/download/${tag_name}/mailbox-linux-amd64.tar.gz"

echo "Downloading build asset from ${url}"
curl -L "${url}" -o mailbox-linux-amd64.tar.gz

# Replace rather than merge, so a local build left in bin/ can't survive
rm -rf bin
tar -xzf mailbox-linux-amd64.tar.gz --strip-components=1
rm mailbox-linux-amd64.tar.gz

echo "${tag_name}" >"${stamp}"
