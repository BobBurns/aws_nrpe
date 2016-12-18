package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"io/ioutil"
	"log"
)

type Detail struct {
	Host    string
	Time    string
	Service string
	Alert   string
	Value   float64
	Units   string
}

func init() {
	// init cloudwatch session

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		fmt.Println("falied to create session,", err)
		return
	}

	svc = cloudwatch.New(sess)
	svc_ec2 = ec2.New(sess)
}

func main() {
	var hosts []MetricQuery
	data, err := ioutil.ReadFile("thresh.json")
	if err != nil {
		log.Fatalf("readfile: %v", err)
	}
	err = json.Unmarshal([]byte(data), &hosts)
	if err != nil {
		log.Fatalf("unmarshal: %v", err)
	}
	if debug == 1 {
		fmt.Println(hosts)
	}
	// make a map of hostnames to MetricQuery
	var namemap = make(map[string]MetricQuery)
	for i, _ := range hosts {
		namemap[hosts[i].Name] = hosts[i]
	}
	// TODO: handle file directory html/random
	for i, _ := range hosts {

		err := hosts[i].getStatistics("-10m")
		if err != nil {
			log.Printf("Error with getStatistics: %s", err)

		}
	}
	if debug == 1 {
		fmt.Println()
		fmt.Println()
		fmt.Printf("%v", hosts)
	}
}
