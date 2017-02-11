#!/bin/bash
cd "$(dirname $0)/../.."
while true; do
    linescount=$(( RANDOM >> 10 )) # 0->31
    for ((i=0; i<linescount; i++)); do
        echo $RANDOM
    done
    pause=$(( RANDOM >> 13 )) # 0->3
    sleep $pause
done | \
bin/linescount -substring 9 -invert