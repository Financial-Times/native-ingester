# UPP - Native Ingester

Native ingester consumes messages from a Kafka topic (both content and metadata flows) and propagates them to be written by read-write services in Native store accoridngly.

## Code

native-ingester

## Primary URL

https://upp-prod-publishing-glb.upp.ft.com/__native-ingester/

## Service Tier

Platinum

## Lifecycle Stage

Production

## Delivered By

content

## Supported By

content

## Known About By

- hristo.georgiev
- robert.marinov
- elina.kaneva
- tsvetan.dimitrov
- elitsa.pavlova
- ivan.nikolov
- miroslav.gatsanoga
- kalin.arsov
- mihail.mihaylov
- dimitar.terziev

## Host Platform

AWS

## Architecture

Native ingester consumes messages containing native CMS content or native CMS metadata from ONE queue topic. According to the data source, native ingester sends the content or metadata to be written to a specific db collection. Optionally, it forwards consumed messages to a different queue.

## Contains Personal Data

No

## Contains Sensitive Data

No

## Dependencies

- upp-kafka
- nativestorereaderwriter

## Failover Architecture Type

ActivePassive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is deployed in both publishing clusters. The failover guide for the cluster is located here:
https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

Manual failover is needed when a new version of the service is deployed to production. Otherwise, an automated failover is going to take place when releasing.
For more details about the failover process please see: https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster

## Key Management Process Type

Manual

## Key Management Details

To access the service clients need to provide basic auth credentials.
To rotate credentials you need to login to a particular cluster and update varnish-auth secrets.

## Monitoring

- Pub-Prod-EU health:
https://upp-prod-publish-eu.upp.ft.com/__health/__pods-health?service-name=native-ingester-cms
https://upp-prod-publish-eu.upp.ft.com/__health/__pods-health?service-name=native-ingester-metadata
- Pub-Prod-US health:
https://upp-prod-publish-us.upp.ft.com/__health/__pods-health?service-name=native-ingester-cms
https://upp-prod-publish-us.upp.ft.com/__health/__pods-health?service-name=native-ingester-metadata

## First Line Troubleshooting

https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
