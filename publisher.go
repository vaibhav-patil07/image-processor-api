package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	bullpublisher "github.com/kiwfy/golang-bull-publisher"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Pattern string
	Message string
	MessageId string
}


type Options struct {
	QueueName string
}
var Publisher *bullpublisher.Publisher = nil

var publisherOptions Options = Options{
	QueueName: "default",
};

func InitializePublisher(redisUrl string,options Options) error {
	redis := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})
	context := context.Background()
	publisher := &bullpublisher.Publisher{
		Redis:   redis,
		Context: context,
	}
	Publisher = publisher
	publisherOptions = options
	fmt.Println("Publisher initialized successfully")
	return nil
}

func PublishMessage(message Message) error {
	if Publisher == nil {
		return errors.New("publisher not initialized")
	}
	return Publisher.AddJob(publisherOptions.QueueName,message, bullpublisher.Options{
			Attempts:           1,
			Backoff:            0,
			Delay:              0,
			Lifo:               false,
			PreventParsingData: false,
			Priority:           0,
			RemoveOnComplete:   10,
			RemoveOnFail:       1,
			Timeout:            0,
			Timestamp:          time.Now().Unix(),
	}, message.MessageId)
}