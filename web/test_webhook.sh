#!/bin/bash

curl -X POST https://800group.silverlininggroup.co.uk/webhook \
  -H "Content-Type: application/json" \
  -H "Timemoto-Signature: testing" \
  -d @./webhooksample.json

secret: tmkey_mFvI8ypVOtSkSsFhDOJ4vyFmPwVf34L6
