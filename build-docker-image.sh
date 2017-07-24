#!/bin/bash

mkdir -p web
cp bootstrap.min.css  bootstrap.min.js index.html  jquery.min.js  jquery-ui.css  jquery-ui.min.js term.js web/
GOARCH=amd64 CGO_ENABLED=0 go build -ldflags -w  -o web/docker-exec-web-console
tar cvf web.tar web/

CURR_PATH=`pwd`

sudo docker build -t docker-web-console:v1.0 $CURR_PATH
