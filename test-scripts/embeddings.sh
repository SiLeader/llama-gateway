#!/usr/bin/env bash

curl http://localhost:8080/v1/embeddings \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "川べりでサーフボードを持った人たちがいます",
    "model": "ruri-v3-310m",
    "encoding_format": "float"
  }'
