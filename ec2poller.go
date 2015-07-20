package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	dataFile = flag.String("file", "store.json", "data store file name")
	status   = flag.String("status", "stopped|pending|terminated", "the status you would like to poll")
)

var store *StatusStore

type Conn struct {
	aw2 *ec2.EC2
}

// create a struct to map toml config file
// TODO: add this as an switch in init()
type AwsConfig struct {
	AwsSecretKey string `toml:"AWS_ACCESS_KEY_ID"`
	AwsAccessKey string `toml:"AWS_SECRET_ACCESS_KEY"`
	Region       string `toml:"AWS_REGION"`
}

func recieveStatus(dataMap map[string]string) <-chan string {
	c := make(chan string)
	go func() {
		for _, v := range dataMap {
			c <- fmt.Sprintf("%s", v)
			//			time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)
		}
	}()
	return c

}

func (c *Conn) iterateResToMap(resp *ec2.DescribeInstancesOutput) map[string]string {
	insMap := make(map[string]string)
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			// fmt.Printf("   Instance State: %v InstanceID: %v \n", *inst.State.Name, *inst.InstanceID)
			// dereference pointer
			var id, state string
			id = *inst.PrivateDNSName
			state = *inst.State.Name
			insMap[id] = state
		}
	}
	return insMap
}

func (c *Conn) GetEc2Data() map[string]string {

	resp, err := c.aw2.DescribeInstances(nil)
	if err != nil {
		log.Fatal(err)
	}

	newMap := c.iterateResToMap(resp)
	return newMap
}

func (c *Conn) startLoop(status string) bool {
	// Call the DescribeInstances Operation
	resp, err := c.aw2.DescribeInstances(nil)
	if err != nil {
		log.Fatal(err)
	}
	// re-format to method call
	newMap := c.iterateResToMap(resp)

	r := recieveStatus(newMap)

	// set a timeout for the channel
	timeout := time.After(5 * time.Second)

	// begin channel operations
	// TODO: this should have interfaces and structs and a poller
	for {
		select {
		case result := <-r:
			matched, _ := regexp.MatchString(status, result)
			if matched {
				res := strings.Split(result, ":")
				status, ip := res[0], res[1]
				fmt.Printf("Status:  %v PrivateIP:  %v  \n", status, ip)
			}
		case <-timeout:
			return false
		}
	}

}

// TODO: right now this pulls creds from env but has the
// capability of using toml the struct is here just need
// to check on if we are using env or fallback to toml
func NewEc2() *Conn {
	c := new(Conn)
	c.aw2 = ec2.New(&aws.Config{Region: "us-west-2"})

	return c
}

func (d *StatusStore) AddDataToFile(status string) {

	for k, v := range d.status {

		if v == status {
			err := d.save(k, v)
			if err != nil {
				log.Printf("something went wrong save %s", k)
			}
		}
	}
}

func main() {
	// Create an EC2 service object in the "us-west-2" region
	// Note that you can also configure your region globally by
	// exporting the AWS_REGION environment variable
	flag.Parse()

	// instantiate new ec2 "object"
	c := NewEc2()

	// Get new Status store
	d := NewStatusStore(*dataFile)

	// Get a data set to work with
	dataSet := c.GetEc2Data()

	// set the status map
	d.status = dataSet

	// lets save some data
	d.AddDataToFile(*status)

}

func Add(value string) {
	key := store.Put(value)
	fmt.Println(key)
}
