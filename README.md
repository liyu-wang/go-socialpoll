# go-socialpoll

## init workspace

go work init

## init module and add dependencies

mkdir chatvotes
cd chatvotes

go mod init github.com/liyu-wang/go-socialpoll/chatvotes

go get github.com/nsqio/go-nsq@latest

go get go.mongodb.org/mongo-driver

## add chatvotes module to workspace

go work use -r ./chatvotes

## start nsq and mongodb

nsqlookupd

nsqd --lookupd-tcp-address=localhost:4160

mongod --dbpath ./db

## Add options to mongodb via mongo shell

> mongosh
> use ballots
> db.polls.insertOne({"title": "Test poll", "options": ["happy", "sad", "fail", "win"]})

## nsq tail to subscribe to topic

> nsq_tail --topic="votes" --lookupd-http-address=localhost:4161

## stop service

> kill -9 $(lsof -t -i:27017)
> sudo lsof -iTCP -sTCP:LISTEN -n -P
> ps aux | grep chatvotes
> kill -9

## init counter module and add dependencies

> mkdir counter
> cd counter
> go mod init github.com/liyu-wang/go-socialpoll/counter
> cd ..
> go work use -r ./counter

go get go.mongodb.org/mongo-driver