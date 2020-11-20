# Topic Manager

Topic Manager is a kubernetes operator to easy manage kafka topics in declarative way.

Defining a Kafka Cluster on the kubernetes cluster
```yaml
apiVersion: broker.bcandido.com/v1alpha1
kind: Broker
metadata:
  name: kafka-development
spec:
  type: kafka
  configuration:
    bootstrapServers:
      - kafka-1:9092
      - kafka-2:9092
      - kafka-3:9092
```

Creating Topics:
```yaml
apiVersion: broker.bcandido.com/v1alpha1
kind: Topic
metadata:
  name: events.status-conclued
spec:
  broker: kafka-development
  configuration:
    partitions: 15
    replicationFactor: 3
```