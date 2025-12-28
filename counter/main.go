package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var fatalErr error

func fatal(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	fatalErr = e
}

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

	pollData := client.Database("ballets").Collection("polls")
}
