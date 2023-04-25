#!/bin/bash

if [[ $# -ne 2 ]]; then 
    echo "usage: run.sh <path>"
    exit 1
fi

export GRAPH_PATH=$1
flask --app viz.py run --host=0.0.0.0