#!/bin/bash

curl -X POST http://localhost:8080/webhook \
     -H "Content-Type: application/json" \
     -H "Timemoto-Signature: testing" \
     -d @/mnt/c/Users/james/code/group800/web/webhooksample.json
