#!/bin/bash

if [ -d "fuzz_cov" ]; then
    rm -rf fuzz_cov
fi

if [ -d "results" ]; then
    rm -rf results
fi

mkdir fuzz_cov
export GOCOVERDIR=fuzz_cov
./raft-fuzzing compare -e 10000 --runs 5
