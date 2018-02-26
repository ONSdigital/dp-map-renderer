#!/bin/bash

if [[ $(docker inspect --format="{{ .State.Running }}" dp-map-renderer) == "false" ]]; then
  exit 1;
fi
