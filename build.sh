#!/bin/sh
cd $(dirname $0)

function usage {
	echo "Usage: $0 [ target ], with optional target = windows|linux|macos, defaults to local platform"
}

if [ $# -gt 1 ]; then
	usage
	exit 1
fi
tgt=bin
if [ $# -eq 1 ]; then
	case "$1" in
	windows)
	GOOS=windows
	;;
	linux)
	GOOS=linux
	;;
	macos)
	GOOS=darwin
	;;
	*)
	;;
	esac
	GOARCH=amd64 # defaults to 64bits!
	tgt="$tgt/${GOOS}_${GOARCH}"
	export GOOS GOARCH
	echo "Building for ${GOOS}_${GOARCH} architecture"
else
	echo "Building for default(local) architecture"
fi
mkdir -p "$tgt"

GOPATH="$(pwd)"
opt=
#opt="$opt -x" # verbose

for tool in src/cmd/*; do
    name="$(basename "$tool")"
    echo Building $name
    go build $opt -o "$tgt/$name" $tool/main.go
done
echo Done.
