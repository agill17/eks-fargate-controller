package controllers

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func createFProfile(input *eks.CreateFargateProfileInput, eksClient eksiface.EKSAPI) error {

	//TODO: add check to make sure subnets exists -- would require the creds to have permissions
	//TODO: add check to make sure roleArn exists -- would require the creds to have permissions

	if _, errCreatingFargateProfile := eksClient.CreateFargateProfile(input); errCreatingFargateProfile != nil {
		return errCreatingFargateProfile
	}

	return nil
}
