#!/bin/sh
# go test doesn't play well with noexec /tmp
sudo mkdir /gotmp
sudo mount tmpfs -t tmpfs /gotmp
TMPDIR=/gotmp go test -v -race
sudo umount /gotmp
sudo rm -rf /gotmp
