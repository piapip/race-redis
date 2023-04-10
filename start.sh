#!/bin/bash

docker-compose --file docker/Redis/docker-compose-redis-replicas.yml down
img=$(docker ps -a -q)
if [ -n "$img" ]; then
    echo "Removing dangling images"
    docker rm -f $img
fi;
vol=$(docker volume ls -q)
if [ -n "$vol" ]; then
    echo "Removing unused volumes"
    docker volume rm $(docker volume ls -q)
fi;
docker network rm redis-default
docker-compose --file ./docker-compose.yaml build
docker-compose --file ./docker-compose.yaml up -d
