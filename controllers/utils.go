package controllers

import (
	kafkamanager "github.com/bcandido/topic-controller"
	brokerv1alpha1 "github.com/bcandido/topic-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"
)

const (
	second = time.Duration(1000000000)
)

func buildTopicController(broker *brokerv1alpha1.Broker) (kafkamanager.TopicControllerAPI, error) {
	topicController := getKafkaTopicController(broker.ConnectionString())
	if topicController == nil {
		return topicController, errors.NewServiceUnavailable("error creating kafka client")
	}
	return topicController, nil
}

func getKafkaTopicController(bootstrapServers string) kafkamanager.TopicControllerAPI {
	kafkaConfig := kafkamanager.KafkaConfig{Brokers: bootstrapServers}
	topicController := kafkamanager.New(kafkaConfig)
	return topicController
}
