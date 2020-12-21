package controllers

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func createFProfile(input *eks.CreateFargateProfileInput, eksClient eksiface.EKSAPI) error {

	if _, errCreatingFargateProfile := eksClient.CreateFargateProfile(input); errCreatingFargateProfile != nil {
		return errCreatingFargateProfile
	}

	return nil
}
