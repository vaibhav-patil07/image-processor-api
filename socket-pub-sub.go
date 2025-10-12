package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/gorilla/websocket"
	"encoding/json"
)

var redisEventSubscriber *redis.Client = nil
var clients = make(map[string]*websocket.Conn)

type ImageProcessorProgressMessage struct {
	ImageID string `json:"image_id"`
	UserId string `json:"user_id"`
	Filename string `json:"filename"`
	Progress int `json:"progress"`
	Status string `json:"status"`
}

func InitializeEventSubscriber(redisUrl string) error {
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return err
	}
	redis := redis.NewClient(opt)
	redisEventSubscriber = redis
	pong, err := redis.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	fmt.Println("Redis Event Subscriber connected successfully", pong)
	return nil
}

func CloseEventSubscriber() {
	redisEventSubscriber.Close()
	redisEventSubscriber = nil
}

func SubscribeToEvent(event string) {
	pubsub := redisEventSubscriber.Subscribe(context.Background(), event)
	defer pubsub.Close()
	for msg := range pubsub.Channel() {
		var imageProcessorProgressMessage ImageProcessorProgressMessage
		err := json.Unmarshal([]byte(msg.Payload), &imageProcessorProgressMessage)
		if err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}
		client := clients[imageProcessorProgressMessage.UserId]
		if client == nil {
			continue
		}
		client.WriteJSON(imageProcessorProgressMessage)
	}

}