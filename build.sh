#!/bin/sh
cd $(dirname $0)
pwd
GOPATH="$(pwd)"
mkdir -p bin
opt=
#opt="$opt -x" # verbose
for tool in src/cmd/*; do
    name="$(basename "$tool")"
    echo Building $name
    go build $opt -o bin/$name $tool/main.go
done
echo Done.