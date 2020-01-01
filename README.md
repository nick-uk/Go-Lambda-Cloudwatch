# Go demo with AWS SDK

## What is the repository for?
#### This is a quick Lambda function PoC to work as REST API with API Gateway for demo and performance comparison reasons.

> Generates a JSON output with CPU load peak, Networking IN peak activity and averages/total calculation for the last 3 days for anomaly detection tasks. 

## Results:
#### I got the expected better results using go routines:
- Sync calls: Duration: 1644.97 ms, Billed Duration: 1700 ms, Max Memory Used: 54 MB, Init Duration: 93.69 ms
- Async calls: Duration: 828.66 ms, Billed Duration: 900 ms, Max Memory Used: 42 MB, Init Duration: 93.69 ms

> Seems possible to save a lot of time and money using Go for lambda. 
> See Python implementation comparison at: https://github.com/nick-uk/Python-Lambda-Cloudwatch/blob/master/README.md
