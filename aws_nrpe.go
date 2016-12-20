package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const debug int = 0

var svc *cloudwatch.CloudWatch
var svc_ec2 *ec2.EC2

type Dimension struct {
	DimName  string `json:"dim_name"`
	DimValue string `json:"dim_value"`
}
type QueryResult struct {
	Alert string
	Units string
	Value float64
	Time  float64
}

type MetricQuery struct {
	Name       string      `json:"name"`
	Host       string      `json:"hostname"`
	Namespace  string      `json:"namespace"`
	Dims       []Dimension `json:"dimensions"`
	Label      string      `json:"metric"`
	Statistics string      `json:"statistics"`
	Warning    string      `json:"warning"`
	Critical   string      `json:"critical"`
	Results    []QueryResult
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

// sort functions
type ByTime []QueryResult

func (a ByTime) Len() int {
	return len(a)
}
func (a ByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByTime) Less(i, j int) bool {
	return a[i].Time < a[j].Time
}

func (mq *MetricQuery) getStatistics(timeframe string) error {

	t := time.Now()
	if mq.Namespace == "AWS/S3" {
		timeframe = "-36h"
	}
	duration, _ := time.ParseDuration(timeframe)
	s := t.Add(duration)
	var dims []*cloudwatch.Dimension
	for i := 0; i < len(mq.Dims); i++ {
		dims = append(dims, &cloudwatch.Dimension{
			Name:  aws.String(mq.Dims[i].DimName),
			Value: aws.String(mq.Dims[i].DimValue),
		})
	}
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(t),
		Namespace:  aws.String(mq.Namespace),
		Period:     aws.Int64(360),
		StartTime:  aws.Time(s),
		Dimensions: dims,
		MetricName: aws.String(mq.Label),
		Statistics: []*string{
			aws.String(mq.Statistics),
		},
	}
	resp, err := svc.GetMetricStatistics(&params)
	if err != nil {
		return fmt.Errorf("Metric query failed: %s", err.Error())
	}
	if len(resp.Datapoints) == 0 {
		if debug == 1 {
			fmt.Println("no datapoints")
		}

		data := QueryResult{
			Value: 0.0,
			Units: "Unknown",
			Time:  float64(time.Now().Unix()),
			Alert: "Unknown",
		}
		mq.Results = append(mq.Results, data)
		return nil
	}
	for _, dp := range resp.Datapoints {
		unit := *dp.Unit
		value := 0.0
		switch mq.Statistics {
		case "Maximum":
			value = *dp.Maximum
		case "Average":
			value = *dp.Average
		case "Sum":
			value = *dp.Sum
		case "SampleCount":
			value = *dp.SampleCount
		case "Minimum":
			value = *dp.Minimum
		}

		if unit == "Bytes" {
			if value > 1048576.0 {
				value = value / 104857.0
				unit = "MB"
			} else if value > 1028.0 {
				value = value / 1028.0
				unit = "KB"
			}
		}

		data := QueryResult{
			Value: value,
			Units: unit,
			Time:  float64(dp.Timestamp.Unix()),
		}
		if test := compareThresh(value, mq.Critical); test {
			data.Alert = "Critical"
		} else if test = compareThresh(value, mq.Warning); test {
			data.Alert = "Warning"
		} else {
			data.Alert = "OK"
		}
		// fix		data.compareThresh(mq.Warning, mq.Critical)
		mq.Results = append(mq.Results, data)
	}

	sort.Sort(ByTime(mq.Results))
	if debug == 1 {
		fmt.Printf("Get Statistics Result: %v", mq)
	}

	return nil
}

func checkerr(e error) {
	if e != nil {
		panic(e)
		log.Fatalf("error: %v", e)
	}
}
func compareThresh(val float64, thresh string) bool {

	thresh = strings.Replace(thresh, "~", "", -1)
	if match, _ := regexp.MatchString("^\\.[0-9]+?", thresh); match {
		thresh = "0" + thresh
	}
	r1, _ := regexp.Compile("^[0-9]+(\\.[0-9]+)?$")                     //match range 10
	r2, _ := regexp.Compile("^[0-9]+(\\.[0-9]+)?:$")                    //match range 10:
	r3, _ := regexp.Compile("^:[0-9]+(\\.[0-9]+)?$")                    //match range :10
	r4, _ := regexp.Compile("^[0-9]+(\\.[0-9]+)?:[0-9]+(\\.[0-9]+)?$")  //match range 20:10
	r5, _ := regexp.Compile("^@[0-9]+(\\.[0-9]+)?:[0-9]+(\\.[0-9]+)?$") //match range @20:10

	if r1.MatchString(thresh) {
		x, err := strconv.ParseFloat(thresh, 64)
		checkerr(err)
		if val < 0 || val > x {
			return true
		}
	} else if r2.MatchString(thresh) {
		x, err := strconv.ParseFloat(thresh[:len(thresh)-1], 64)
		checkerr(err)
		if val < x {
			return true
		}
	} else if r3.MatchString(thresh) {
		x, err := strconv.ParseFloat(thresh[1:], 64)
		checkerr(err)
		if val > x {
			return true
		}
	} else if r4.MatchString(thresh) {
		values := strings.Split(thresh, ":")
		x, err := strconv.ParseFloat(values[0], 64)
		checkerr(err)
		y, err := strconv.ParseFloat(values[1], 64)
		checkerr(err)

		if val < x || val > y {
			return true
		}
	} else if r5.MatchString(thresh) {
		values := strings.Split(thresh, ":")
		x, err := strconv.ParseFloat(values[0][1:], 64)
		checkerr(err)
		y, err := strconv.ParseFloat(values[1], 64)
		checkerr(err)

		if val >= x && val <= y {
			return true
		}
	}
	return false
}

func main() {
	// map output string to exit code
	var exitcond = make(map[string]int)
	exitcond["Warning"] = 1
	exitcond["Critical"] = 2
	exitcond["Unknown"] = 3
	exitcond["OK"] = 0

	// parse service configuration file
	var services []MetricQuery
	data, err := ioutil.ReadFile("thresh.json")
	if err != nil {
		log.Fatalf("readfile: %v", err)
	}
	err = json.Unmarshal([]byte(data), &services)
	if err != nil {
		log.Fatalf("unmarshal: %v", err)
	}
	if debug == 1 {
		fmt.Println(services)
	}
	// make a map of hostnames to MetricQuery
	var serviceMap = make(map[string]MetricQuery)
	for i, _ := range services {
		serviceMap[services[i].Name] = services[i]
	}

	// get command line arguments
	servPtr := flag.String("service", "required", "User defined service name. Required!")
	warnPtr := flag.String("w", "use config", "nagios spec warning value range")
	critPtr := flag.String("c", "use config", "nagios spec critical value range")
	flag.Parse()

	if *servPtr == "required" {
		fmt.Println("Please provide service name. Usage: ")
		flag.PrintDefaults()
		os.Exit(4)
	}
	query, ok := serviceMap[*servPtr]
	if ok == false {
		fmt.Printf("No service '%s' found. Check thresh.json for user defined services.\n", os.Args[1])
		os.Exit(3)
	}
	if *warnPtr != "use config" {
		query.Warning = *warnPtr
	}
	if *critPtr != "use config" {
		query.Critical = *critPtr
	}

	// query aws cloudwatch

	err = query.getStatistics("-10m")
	if err != nil {
		log.Fatalf("Error with getStatistics: %s", err)
	}

	// print perfdata and exit nagios status
	last := len(query.Results) - 1
	fmt.Printf("%s %s | %s=%f\n", query.Label, query.Results[last].Alert, query.Label, query.Results[last].Value)

	os.Exit(exitcond[query.Results[last].Alert])
}
