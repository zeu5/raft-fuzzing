#!/bin/bash

if [ -d "fuzz_cov" ]; then
    rm -rf fuzz_cov
fi

mkdir fuzz_cov
export GOCOVERDIR=fuzz_cov
./raft-fuzzing compare -e 10 --runs 1