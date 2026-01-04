# go-socialpoll

## init workspace

``` bash
go work init
```

## init module and add dependencies

``` bash
mkdir chatvotes
cd chatvotes

go mod init github.com/liyu-wang/go-socialpoll/chatvotes

go get github.com/nsqio/go-nsq@latest
go get go.mongodb.org/mongo-driver
```

## add chatvotes module to workspace

``` bash
go work use -r ./chatvotes
```

## db design

```json
{ 
  "_id": "???", 
  "title": "Poll title", 
  "options": ["one", "two", "three"], 
  "results": { 
    "one": 100, 
    "two": 200, 
    "three": 300 
  } 
}
```

## start nsq and mongodb

``` bash
nsqlookupd
nsqd --lookupd-tcp-address=localhost:4160
mongod --dbpath ./db
```

## Add options to mongodb via mongo shell (do this before start the services)

``` bash
mongosh
use ballots
db.polls.insertOne({"title": "Test poll", "options": ["happy", "sad", "fail", "win"]})
```

## nsq tail to subscribe to topic

``` bash
nsq_tail --topic="votes" --lookupd-http-address=localhost:4161
```

## stop service by ports

``` bash
kill -9 $(lsof -t -i:27017)
sudo lsof -iTCP -sTCP:LISTEN -n -P
ps aux | grep chatvotes
kill -9 <pid>
```

## init counter module and add dependencies

``` bash
mkdir counter
cd counter
go mod init github.com/liyu-wang/go-socialpoll/counter
cd ..
go work use -r ./counter

go get go.mongodb.org/mongo-driver
```

## init api module

``` bash
mkdir api
cd api
go mod init github.com/liyu-wang/go-socialpoll/api
cd ..
go work use -r ./api
```

## verify db update

``` bash
mongosh
use ballots
db.polls.find().pretty()
```

## verify api with curl

``` bash
curl -X GET http://localhost:8080/polls/ \
  -H "X-API-Key: abc123"

curl -X GET http://localhost:8080/polls/6955b7f4cf53b12a54c2b11b \
  -H "X-API-Key: abc123"

curl --data '{"title":"test","options":["one","two","three"]}' \
  -X POST http://localhost:8080/polls/ \
  -H "X-API-Key: abc123"

curl -X DELETE http://localhost:8080/polls/695a4a4a76f401f82ada14ca \
  -H "X-API-Key: abc123"
```

## start service

```bash
./start-services.sh
```

This will open new Terminal windows for each service:

- nsqlookupd
- nsqd
- MongoDB
- chatvotes application
- counter application

## stop service

Use the stop script:

```bash
./stop-services.sh
```

Or manually:

```bash
pkill nsqlookupd
pkill nsqd
pkill mongod
pkill -f "chatvotes"
pkill -f "counter"
```

Or by PID:

```bash
kill -9 $(lsof -t -i:27017)  # MongoDB
kill -9 $(lsof -t -i:4150)   # NSQd
kill -9 $(lsof -t -i:4160)   # NSQLookupd
```
