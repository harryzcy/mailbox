#!/bin/bash

tag_name=$(
    curl -s https://api.github.com/repos/harryzcy/mailbox/releases/latest |
        grep "tag_name" |
        cut -d : -f 2,3 |
        tr -d "\",[:space:]"
)

url="https://github.com/harryzcy/mailbox/releases/download/${tag_name}/mailbox-linux-amd64.tar.gz"

echo "Downloading build asset from ${url}"
curl -L "${url}" -o mailbox-linux-amd64.tar.gz

tar -xzvf mailbox-linux-amd64.tar.gz --strip-components=1
rm mailbox-linux-amd64.tar.gz
