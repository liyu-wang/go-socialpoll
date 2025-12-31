package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var fatalErr error

func fatal(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	fatalErr = e
}

const updateDuration = 1 * time.Second

func main() {
	defer func() {
		if fatalErr != nil {
			os.Exit(1)
		}
	}()

	log.Println("Connecting to database...")

	// Connection context with timeout for initial connection only
	connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connCancel()

	uri := "mongodb://localhost:27017"
	client, err := mongo.Connect(connCtx, options.Client().ApplyURI(uri))
	if err != nil {
		fatal(fmt.Errorf("failed to connect to mongodb: %w", err))
		return
	}

	// Register cleanup BEFORE ping - ensures it always runs
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := client.Disconnect(disconnectCtx); err != nil {
			fatal(fmt.Errorf("failed to disconnect from mongodb: %w", err))
		}
		cancel()
		log.Println("Disconnected from mongodb")
	}()

	// Verify connection with a separate timeout context
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Ping(pingCtx, nil)
	cancel()
	if err != nil {
		fatal(fmt.Errorf("failed to ping mongodb: %w", err))
		return
	}
	log.Println("Successfully connected to mongodb")

	// Create a separate, non-expiring context for all database operations
	operationCtx, operationCancel := context.WithCancel(context.Background())
	defer operationCancel()

	pollData := client.Database("ballots").Collection("polls")

	var counts map[string]int
	var countsLock sync.Mutex

	log.Println("Connecting to nsq...")
	q, err := nsq.NewConsumer("votes", "counter", nsq.NewConfig())
	if err != nil {
		fatal(fmt.Errorf("failed to create nsq consumer: %w", err))
		return
	}

	q.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		countsLock.Lock()
		defer countsLock.Unlock()
		if counts == nil {
			counts = make(map[string]int)
		}
		vote := string(message.Body)
		counts[vote]++
		log.Printf("Vote received: %s, total: %d\n", vote, counts[vote])
		return nil
	}))

	// Connect to nsqlookupd
	if err := q.ConnectToNSQLookupd("localhost:4161"); err != nil {
		fatal(fmt.Errorf("failed to connect to nsq: %w", err))
		return
	}

	// Periodic timer to update database with vote counts
	ticker := time.NewTicker(updateDuration)
	termchan := make(chan os.Signal, 1)
	signal.Notify(termchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Keep the application running
	for {
		select {
		case <-ticker.C:
			doCount(operationCtx, &countsLock, &counts, pollData)
		case <-termchan:
			ticker.Stop()
			q.Stop()
		case <-q.StopChan:
			//finished
			return
		}
	}
}

func doCount(ctx context.Context, countsLock *sync.Mutex, counts *map[string]int, pollData *mongo.Collection) {
	countsLock.Lock()
	defer countsLock.Unlock()

	if len(*counts) == 0 {
		log.Println("No new votes, skipping database update")
		return
	}

	log.Println("Updating database...")
	log.Println("Current counts:", *counts)

	ok := true
	for option, count := range *counts {
		// Create a dedicated timeout context for this operation
		opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		// filter to find the poll option
		sel := bson.M{"options": bson.M{"$in": []string{option}}}
		log.Printf("Searching with filter: %v", sel)

		// update to increment the vote count
		up := bson.M{"$inc": bson.M{"results." + option: count}}
		result, err := pollData.UpdateMany(opCtx, sel, up)
		if err != nil {
			log.Printf("Error updating vote count for %s: %v", option, err)
			ok = false
		} else {
			log.Printf("Updated %d documents for option '%s' with count %d", result.ModifiedCount, option, count)
		}
		cancel()
	}
	if ok {
		log.Println("Finished updating database...")
		*counts = nil // reset counts after successful update
	}
}
