#!/usr/bin/env bash

curl http://localhost:8080/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gemma3-270m",
    "input": "What is the capital of Tokyo?"
  }'
