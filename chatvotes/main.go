package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Entry point for the chatvotes application
	var stopFlag atomic.Bool
	stopchan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)

	// goroutine to handle interrupt signal
	go func() {
		// wait for an interrupt signal such as control+C
		<-signalChan
		stopFlag.Store(true)
		log.Println("Stopping...")
		stopchan <- struct{}{}
		closeWSConn()
	}()

	// send the interrupt signal to signalChan
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// connect to the database
	if err := dialdb(); err != nil {
		log.Println("warning: failed to dial mongodb:", err)
		log.Println("continuing without database...")
	}
	defer closedb()

	// start things
	votes := make(chan string)
	publisherStoppedChan := publishVotes(votes)
	chatStoppedChan := startChatStream(stopchan, votes)

	// periodic closer to force reconnects
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			log.Println("periodic closer: closing websocket connection to force reconnect")
			closeWSConn()
			log.Println("periodic closer: done closing websocket connection")
			if stopFlag.Load() {
				return
			}
		}
	}()

	<-chatStoppedChan
	close(votes)
	<-publisherStoppedChan
	log.Println("Stopped.")
}

var dbClient *mongo.Client
var dbCtx context.Context
var dbCancel context.CancelFunc
var operationCtx context.Context
var operationCancel context.CancelFunc

func dialdb() error {
	log.Println("dialing mongodb: localhost")

	// Connection context with timeout for initial connection only
	dbCtx, dbCancel = context.WithTimeout(context.Background(), 10*time.Second)

	uri := "mongodb://localhost:27017"
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	// Verify connection with a separate timeout context
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Ping(pingCtx, nil)
	cancel()
	if err != nil {
		client.Disconnect(context.Background())
		return err
	}

	// Create a separate, non-expiring context for all database operations
	// Only after we know the connection is valid
	operationCtx, operationCancel = context.WithCancel(context.Background())

	dbClient = client
	log.Println("successfully connected to mongodb")
	return nil
}

func closedb() {
	// Cancel all pending operations first
	if operationCancel != nil {
		operationCancel()
	}

	if dbClient != nil {
		// Use a fresh timeout context for disconnect (not the expired dbCtx)
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := dbClient.Disconnect(disconnectCtx); err != nil {
			log.Println("error closing mongodb connection:", err)
		} else {
			log.Println("closed mongodb connection")
		}
		cancel()
	}

	if dbCancel != nil {
		dbCancel()
	}
}

type poll struct {
	Options []string `bson:"options"`
}

func loadOptions() ([]string, error) {
	if dbClient == nil {
		log.Println("warning: database not connected, returning empty options")
		return []string{}, nil
	}

	// Create a dedicated timeout context for this operation
	// Each call to loadOptions gets its own timeout
	ctx, cancel := context.WithTimeout(operationCtx, 5*time.Second)
	defer cancel()

	collection := dbClient.Database("ballots").Collection("polls")

	// Use empty bson.M{} instead of nil for clarity
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Println("error finding polls:", err)
		return []string{}, nil
	}
	defer cursor.Close(ctx)

	// Use cursor.All() for cleaner code
	var polls []poll
	if err = cursor.All(ctx, &polls); err != nil {
		log.Println("error decoding polls:", err)
		return []string{}, nil
	}

	var options []string
	for _, p := range polls {
		options = append(options, p.Options...)
	}

	if len(options) == 0 {
		log.Println("no poll options found in database")
	} else {
		log.Printf("loaded %d poll options\n", len(options))
	}

	return options, nil
}

func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)
	pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
	go func() {
		for vote := range votes {
			pub.Publish("votes", []byte(vote)) // publish vote to NSQ
			log.Println("Published vote:", vote)
		}
		log.Println("Publisher: stopping")
		pub.Stop()
		log.Println("Publisher: stopped")
		stopchan <- struct{}{}
	}()
	return stopchan
}
