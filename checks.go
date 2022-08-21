package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"golang.org/x/crypto/ssh"
)

func blockUntilNotPending(svc *ec2.Client, id string) (string, error) {
	for {
		res, err := svc.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
			InstanceIds: []string{id},
		})
		if err != nil {
			return "", err
		}

		state := res.Reservations[0].Instances[0].State.Name
		if res.Reservations[0].Instances[0].State.Name != types.InstanceStateNamePending {
			return fmt.Sprintf("%s", state), nil
		}
	}
}

func blockUntilSsh(svc *ec2.Client, id string) error {
	res, err := svc.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return err
	}
	ip := *res.Reservations[0].Instances[0].PublicDnsName

	log.Printf("IP: %s\n", ip)

	pem, err := os.ReadFile(*pemFile)
	if err != nil {
		return err
	}
	signer, err := ssh.ParsePrivateKey(pem)

	cfg := ssh.ClientConfig{
		Config:          ssh.Config{},
		User:            *sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second * 60,
	}

	for {
		client, err := ssh.Dial("tcp", ip+":22", &cfg)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				time.Sleep(time.Second)
				continue
			}
			return err
		} else {
			client.Close()
			return nil
		}
	}
}
