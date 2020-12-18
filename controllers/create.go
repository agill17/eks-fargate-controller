package controllers

import (
	"github.com/agill17/eks-fargate-controller/api/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createFProfile(cr *v1alpha1.FargateProfile, eksClient eksiface.EKSAPI, client client.Client) error {

	//TODO: add check to make sure subnets exists -- would require the creds to have permissions
	//TODO: add check to make sure roleArn exists -- would require the creds to have permissions

	createInput := &eks.CreateFargateProfileInput{
		ClusterName:         aws.String(cr.Spec.ClusterName),
		FargateProfileName:  aws.String(cr.GetName()),
		PodExecutionRoleArn: aws.String(cr.Spec.PodExecutionRoleArn),
		Subnets:             aws.StringSlice(cr.Spec.Subnets),
		Tags:                aws.StringMap(cr.Spec.Tags),
	}

	var selectors []*eks.FargateProfileSelector

	if len(cr.Spec.PodSelectors) > 0 {
		for key, val := range cr.Spec.PodSelectors {
			selectors = append(selectors, &(eks.FargateProfileSelector{
				Labels:    aws.StringMap(map[string]string{key: val}),
				Namespace: aws.String(cr.GetNamespace()),
			}))
		}
	}
	createInput.SetSelectors(selectors)

	if _, errCreatingFargateProfile := eksClient.CreateFargateProfile(createInput); errCreatingFargateProfile != nil {
		return errCreatingFargateProfile
	}

	return updateCrPhase(v1alpha1.Creating, client, cr)
}
