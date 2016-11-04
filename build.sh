#! /bin/sh
go build -ldflags "-X main.Version=$(cat ./version)" -o s3r s3restore.go
