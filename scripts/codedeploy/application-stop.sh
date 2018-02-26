#!/bin/bash

CONTAINER_ID=$(docker ps | grep dp-map-renderer | awk '{print $1}')

if [[ -n $CONTAINER_ID ]]; then
  docker stop $CONTAINER_ID
fi
