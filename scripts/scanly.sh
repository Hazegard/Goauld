#!/bin/bash

unzip goauld.zip
cd goauld || return
sudo docker image load -i goauld_server.img
sudo /usr/local/bin/docker-compose up -d --force-recreate && sudo /usr/local/bin/docker-compose logs -f --tail 1000