#!/bin/bash

if ! sudo systemctl is-active --quiet docker.service; then
    sudo systemctl start docker.service
fi

sudo docker image prune -f
sudo docker images --format '{{.Repository}}:{{.Tag}}' | grep 'lemonai' | xargs -r sudo docker rmi

sudo docker build -t lemonai .

imageName="$1"
sudo docker tag lemonai:latest $imageName
sudo docker push $imageName