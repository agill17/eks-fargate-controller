package controllers

import (
	"github.com/agill17/eks-fargate-controller/api/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deleteFprofile(cr *v1alpha1.FargateProfile, eksClient eksiface.EKSAPI, client client.Client) error {

	if errMarkingFpDeleting := updateCrPhase(v1alpha1.Deleting, client, cr); errMarkingFpDeleting != nil {
		return errMarkingFpDeleting
	}

	if _, errDeleting := eksClient.DeleteFargateProfile(&eks.DeleteFargateProfileInput{
		ClusterName:        aws.String(cr.Spec.ClusterName),
		FargateProfileName: aws.String(cr.GetName()),
	}); errDeleting != nil {
		// remove finalizer if it does not exist on AWS side
		if awsErr, ok := errDeleting.(awserr.Error); ok && awsErr.Code() == eks.ErrCodeResourceNotFoundException {
			return RemoveFinalizer(FargateProfileFinalizer, cr, client)
		}
		return errDeleting
	}

	return RemoveFinalizer(FargateProfileFinalizer, cr, client)
}
