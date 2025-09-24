package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/ktbsomen/gobullmq"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Pattern string `json:"pattern"`
	Message string `json:"message"`
	MessageId string `json:"message_id"`
}

type Options struct {
	QueueName string
}

var messageQueue *gobullmq.Queue = nil
var redisClient *redis.Client = nil

func InitializePublisher(redisUrl string,options Options) error {
	redis := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})
	redisClient = redis
	pong, err := redis.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	fmt.Println("Redis connected successfully", pong)
	context := context.Background()

	queue ,err:= gobullmq.NewQueue(context, options.QueueName, redis);
	if err != nil {
		return err
	}
	messageQueue = queue
	fmt.Println("Publisher initialized successfully")
	return nil
}

func PublishMessage(message Message) error {
	if messageQueue == nil {
		return errors.New("publisher not initialized")
	}
	_,err := messageQueue.Add(context.Background(), message.Pattern,message)
	if err != nil {
		return err
	}
	return nil
}

func PingPublisher() error {
	if messageQueue == nil {
		return errors.New("publisher not initialized")
	}
	_,err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	return nil
}