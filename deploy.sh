#!/bin/bash

if ! [ -f src/localhost.go.config ];then echo localhost.go.config must exist in src; exit 1;fi
if ! [ -f src/production.go.config ];then echo production.go.config must exist in src; exit 1;fi
if ! [ x"$1" != x ];then echo first argument must be your application id; exit 1;fi

ln -sf production.go.config src/config.go
goapp deploy -application "$1" -version dev -oauth app.yaml
ln -sf localhost.go.config src/config.go
