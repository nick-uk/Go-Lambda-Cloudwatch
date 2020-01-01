package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type cpuMax struct {
	Perc float64   `json:"perc"`
	Time time.Time `json:"time"`
	Avg  float64   `json:"avg"`
}

type netPeak struct {
	Max   float64   `json:"max"`
	Time  time.Time `json:"time"`
	Total float64   `json:"total"`
}

type body struct {
	CPU cpuMax  `json:"cpu"`
	Net netPeak `json:"net"`
}

type lambdaRes struct {
	Body       string `json:"body"`
	StatusCode int    `json:"statuscode"`
}

func getCPU(client *cloudwatch.Client) cpuMax {

	req := client.GetMetricStatisticsRequest(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []cloudwatch.Dimension{
			cloudwatch.Dimension{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String("managers-ag"),
			},
		},
		StartTime: aws.Time(time.Now().Add(-(time.Hour * 24 * 3))),
		EndTime:   aws.Time(time.Now()),
		Period:    aws.Int64(300),
		Statistics: []cloudwatch.Statistic{
			"Average",
		},
		Unit: "Percent",
	})

	resp, err := req.Send(context.Background())
	if err != nil {
		panic("failed, " + err.Error())
	}

	//fmt.Println("Response", *resp.Datapoints[0].Average)
	gavg := 0.0
	max := cpuMax{}

	for i := range resp.Datapoints {
		gavg += *resp.Datapoints[i].Average
		if *resp.Datapoints[i].Average > max.Perc {
			max.Perc = *resp.Datapoints[i].Average
			max.Time = *resp.Datapoints[i].Timestamp
		}
	}

	max.Avg = gavg / float64(len(resp.Datapoints))

	fmt.Println("Managers group CPU MAX:", strconv.FormatFloat(max.Perc, 'f', 0, 64)+"%",
		max.Time.Format("01/02/2006 15:04:05"), ", 3 days AVG:", strconv.FormatFloat(max.Avg, 'f', 0, 64)+"%")

	return max
}

func getNET(client *cloudwatch.Client) netPeak {

	req := client.GetMetricStatisticsRequest(&cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("NetworkIn"),
		Dimensions: []cloudwatch.Dimension{
			cloudwatch.Dimension{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String("managers-ag"),
			},
		},
		StartTime: aws.Time(time.Now().Add(-(time.Hour * 24 * 3))),
		EndTime:   aws.Time(time.Now()),
		Period:    aws.Int64(300),
		Statistics: []cloudwatch.Statistic{
			"Maximum",
		},
		Unit: "Bytes",
	})

	resp, err := req.Send(context.Background())
	if err != nil {
		panic("failed, " + err.Error())
	}

	//fmt.Println("Response", *resp.Datapoints[0].Average)
	max := netPeak{}

	for i := range resp.Datapoints {
		max.Total += *resp.Datapoints[i].Maximum
		if *resp.Datapoints[i].Maximum > max.Max {
			max.Max = *resp.Datapoints[i].Maximum
			max.Time = *resp.Datapoints[i].Timestamp
		}
	}

	fmt.Println("Managers group NET Peak:", strconv.FormatFloat(max.Max/1024, 'f', 0, 64)+" KB",
		max.Time.Format("01/02/2006 15:04:05"), ", 3 days Total:", strconv.FormatFloat(max.Total/1024/1024, 'f', 0, 64)+" MB")

	return max
}

func lambdaHandler() (lambdaRes, error) {

	var cpures cpuMax
	var netres netPeak

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return lambdaRes{}, errors.New("unable to load SDK config, " + err.Error())
	}
	cfg.Region = endpoints.EuWest2RegionID

	client := cloudwatch.New(cfg)

	// sync
	// This gives: Duration: 1644.97 ms, Billed Duration: 1700 ms, Max Memory Used: 54 MB, Init Duration: 93.69 ms
	start := time.Now()
	cpures = getCPU(client)
	netres = getNET(client)
	elapsed := time.Since(start).Seconds()
	fmt.Printf("== Took " + strconv.FormatFloat(elapsed, 'f', 2, 64) + " secs ==\n\n")

	// vs async
	// This gives: Duration: 828.66 ms, Billed Duration: 900 ms, Max Memory Used: 42 MB
	/*
		var cpu = make(chan cpuMax)
		var net = make(chan netPeak)
		start := time.Now()
		go func() {
			cpu <- getCPU(client)
		}()
		go func() {
			net <- getNET(client)
		}()

		for i := 0; i < 2; i++ {
			select {
			case cpures = <-cpu:
			case netres = <-net:
			}
		}
		elapsed := time.Since(start).Seconds()
		fmt.Printf("== Took " + strconv.FormatFloat(elapsed, 'f', 2, 64) + " secs ==\n\n")
	*/

	resBytes, _ := json.Marshal(body{
		cpures,
		netres,
	})
	response := lambdaRes{
		Body:       string(resBytes),
		StatusCode: 200,
	}

	return response, nil
}

func main() {
	if os.Getenv("AWS_EXECUTION_ENV") != "" {
		// Make the handler available for Remote Procedure Call by AWS Lambda
		log.Println("[lambda]", "AWS_EXECUTION_ENV:", os.Getenv("AWS_EXECUTION_ENV"), os.Getenv("AWS_LAMBDA_FUNCTION_VERSION"),
			os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE"))
		lambda.Start(lambdaHandler)
	} else {
		// For local tests
		lambdaHandler()
	}
}
