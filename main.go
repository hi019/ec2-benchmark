package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/pflag"
)

// var nixos = "ami-0a743534fa3e51b41"
// var clearos = "ami-0e8bf6a75bdee4a3c"
var ami = pflag.String("ami", "ami-0e8bf6a75bdee4a3c", "aws ami to use")
var profile = pflag.String("profile", "default", "aws profile to use")
var region = pflag.String("region", "us-east-2", "aws region")
var keyName = pflag.String("key", "benchmark", "the aws key pair name to use for the instance")
var sgId = pflag.String("sgId", "sg-04cbf1db790202599", "the aws security group to apply")
var sshUser = pflag.String("username", "clear", "ssh username")
var pemFile = pflag.String("pem", "", "pem file location")

func main() {
	pflag.Parse()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(*profile), config.WithRegion(*region))
	if err != nil {
		log.Panic(err)
	}
	svc := ec2.New(ec2.Options{Credentials: cfg.Credentials, Region: *region})

	if err := run(svc); err != nil {
		log.Panic(err)
	}
}

func run(svc *ec2.Client) error {
	start := time.Now()

	instance, err := svc.RunInstances(context.Background(), &ec2.RunInstancesInput{
		MaxCount:                          aws.Int32(1),
		MinCount:                          aws.Int32(1),
		ImageId:                           ami,
		InstanceInitiatedShutdownBehavior: types.ShutdownBehaviorTerminate,
		InstanceType:                      types.InstanceTypeT3Micro,
		KeyName:                           keyName,
		SecurityGroupIds:                  []string{*sgId},
	})
	if err != nil {
		return err
	}

	id := *instance.Instances[0].InstanceId
	log.Printf("ID: %s", id)
	defer func() {
		if err := terminate(svc, id); err != nil {
			log.Panic(err)
		}
		log.Printf("Instance terminated. Time: %v\n", time.Now().Sub(start).Seconds())
	}()

	if state, err := blockUntilNotPending(svc, id); err != nil {
		return err
	} else {
		log.Printf("New instance state: %s. Time: %v\n", state, time.Now().Sub(start).Seconds())
	}

	if err := blockUntilSsh(svc, id); err != nil {
		return err
	} else {
		log.Printf("SSH access. Time: %v\n", time.Now().Sub(start).Seconds())
	}

	return nil
}

func terminate(svc *ec2.Client, id string) error {
	_, err := svc.TerminateInstances(context.Background(), &ec2.TerminateInstancesInput{
		InstanceIds: []string{id},
	})
	return err
}
