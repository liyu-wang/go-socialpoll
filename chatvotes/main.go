package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// "gopkg.in/mgo.v2"
	"github.com/globalsign/mgo" // outdated module, but still widely used
	"github.com/nsqio/go-nsq"
)

func main() {
	// Entry point for the chatvotes application
	var stoplock sync.Mutex
	// access from different goroutines
	stop := false
	stopchan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)

	// goroutine to handle interrupt signal
	go func() {
		// wait for an interrupt signal such as contrlol+C
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("Stopping...")
		stopchan <- struct{}{}
		closeWSConn()
	}()

	// send the interrupt signal to signalChan
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// connect to the database
	if err := dialdb(); err != nil {
		log.Fatalln("failed to dial mongodb:", err)
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
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				return
			}
			stoplock.Unlock()
		}
	}()

	<-chatStoppedChan
	close(votes)
	<-publisherStoppedChan
	log.Println("Stopped.")
}

var db *mgo.Session

func dialdb() error {
	var err error
	log.Println("dialing mongodb: localhost")
	db, err = mgo.Dial("localhost")
	return err
}

func closedb() {
	db.Close()
	log.Println("closed mongodb connection")
}

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
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
