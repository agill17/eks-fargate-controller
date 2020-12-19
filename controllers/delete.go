package controllers

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func deleteFprofile(deleteIn *eks.DeleteFargateProfileInput, eksClient eksiface.EKSAPI) error {

	if _, errDeleting := eksClient.DeleteFargateProfile(deleteIn); errDeleting != nil {
		// remove finalizer if it does not exist on AWS side
		if awsErr, ok := errDeleting.(awserr.Error); ok && awsErr.Code() == eks.ErrCodeResourceNotFoundException {
			return nil
		}
		return errDeleting
	}

	return nil
}
