Native Ingester
===============
[![CircleCI](https://circleci.com/gh/Financial-Times/native-ingester.svg?style=svg)](https://circleci.com/gh/Financial-Times/native-ingester) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/native-ingester)](https://goreportcard.com/report/github.com/Financial-Times/native-ingester) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/native-ingester/badge.svg)](https://coveralls.io/github/Financial-Times/native-ingester)

Native ingester implements the following functionality:
1. It consumes messages containing native CMS content or native CMS metadata from ONE queue topic.
2. According to the data source, native ingester writes the content or metadata to a specific db collection.
3. Optionally, it forwards consumed messages to a different queue.

## Installation & running locally

Installation:
```
go get github.com/Financial-Times/native-ingester
cd $GOPATH/src/github.com/Financial-Times/native-ingester
go test ./...
go install

```
Run it locally:
```
$GOPATH/bin/native-ingester
    --read-queue-addresses={source-zookeeper-address}
    --read-queue-group={topic-group}
    --read-queue-topic={source-topic}
    --source-uuid-field='["uuid", "post.uuid", "data.uuidv3"]'
    --native-writer-address={native-writer-address}
```
List all the possible options:
```
$  native-ingester -h

Usage: native-ingester [OPTIONS]

A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue

Options:
  --port="8080"                                 Port to listen on ($PORT)
  --read-queue-addresses=[]                     Zookeeper addresses (host:port) to connect to the consumer queue. ($Q_READ_ADDR)
  --read-queue-group=""                         Group used to read the messages from the queue. ($Q_READ_GROUP)
  --read-queue-topic=""                         The topic to read the messages from. ($Q_READ_TOPIC)
  --native-writer-address=""                    Address (URL) of service that writes persistently the native content ($NATIVE_RW_ADDRESS)
  --config="config.json"                        Configuration file - Mapping from (originId (URI), Content Type) to native collection name, in JSON format, for content_type attribute specify a RegExp Literal expression.
  --content-uuid-fields=[]                      List of JSONPaths that point to UUIDs in native content bodies. e.g. uuid,post.uuid,data.uuidv3 ($NATIVE_CONTENT_UUID_FIELDS)
  --write-queue-address=""                      Kafka address (host:port) to connect to the producer queue. ($Q_WRITE_ADDR)
  --write-topic=""                              The topic to write the messages to. ($Q_WRITE_TOPIC)
  --content-type="Content"                      The type of the content (for logging purposes, e.g. "Content" or "Annotations") the application is able to handle. ($CONTENT_TYPE)
  --appName="native-ingester"                   The name of the application ($APP_NAME)
  --panic-guide=""                              Panic Guide URL ($PANIC_GUIDE_URL)
```

Example command line:

```shell
native-ingester --read-queue-addresses localhost:2181 --read-queue-group nativeIngesterCms --read-queue-topic PreNativeCmsPublicationEvents --native-writer-address http://localhost:8081 --content-uuid-fields uuid --content-uuid-fields post.uuid --content-uuid-fields data.uuidv3 --content-uuid-fields id --write-queue-address localhost:9092 --write-topic NativeCmsPublicationEvents --config config.json --content-type Content --panic-guide https://runbooks.in.ft.com/native-ingester
``` 

If you want to run it with docker-compose and all the services native-ingester depends on, first you need to build them with `local` tag.

Prepare `cms-notifier:local` and `nativerw:local` and then run:

```sh
docker-compose up
```

| Service         | Port |
|-----------------|------|
| native-ingester | 8080 |
| cms-notifier    | 8081 |
| nativerw        | 8083 |


## Admin endpoints

  - `https://{host}/__native-store-{type}/__health`
  - `https://{host}/__native-store-{type}/__gtg`

Note: All API endpoints in CoCo require Authentication.
