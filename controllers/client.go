package controllers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"math"
)

func NewEksClient(region string) eksiface.EKSAPI {
	sess, _ := session.NewSession(&aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(region),
		MaxRetries:                    aws.Int(math.MaxInt64),
	})
	return eks.New(sess)
}
