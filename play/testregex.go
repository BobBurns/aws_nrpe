package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

func checkerr(e error) {
	if e != nil {
		panic(e)
		log.Fatalf("error: %v", e)
	}
}
func comparethresh(val float64, thresh string) bool {

	thresh = strings.Replace(thresh, "~", "", -1)
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
	metricvalues := []float64{10.0, 100.0, 99.99, 80.23, 3.0, 4.0, -23.4}
	thresh := []string{"10", "85:", "~:10.3", "0:100", "@20:100", "3", "23.4"}

	for _, t := range thresh {
		for _, m := range metricvalues {
			fmt.Printf("[%f] in range %s ", m, t)
			result := comparethresh(m, t)
			if result {
				fmt.Printf("is alert\n")
			}
			fmt.Println()
		}
	}
}
