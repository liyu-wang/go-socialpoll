# go-socialpoll

## init workspace

go work init

## init module and add dependencies

mkdir chatvotes
cd chatvotes

go mod init github.com/liyu-wang/go-socialpoll/chatvotes

go get github.com/nsqio/go-nsq@latest

go get go.mongodb.org/mongo-driver/v2/mongo

## add chatvotes module to workspace

go work use -r ./chatvotes

## start nsq and mongodb

nsqlookupd

nsqd --lookupd-tcp-address=localhost:4160

mongod --dbpath ./db

