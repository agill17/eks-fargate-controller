package controllers

import (
	"context"
	"github.com/agill17/eks-fargate-controller/api/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newAwsSession(region string) *session.Session {
	sess, _ := session.NewSession(&aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(region),
		MaxRetries:                    aws.Int(math.MaxInt64),
	})
	return sess
}

func NewEksClient(region string) eksiface.EKSAPI { return eks.New(newAwsSession(region)) }
func NewEc2Client(region string) ec2iface.EC2API { return ec2.New(newAwsSession(region)) }
func NewIamClient(region string) iamiface.IAMAPI { return iam.New(newAwsSession(region)) }

func AddFinalizer(finalizer string, runtimeObj runtime.Object, client client.Client) error {
	metaObj, err := meta.Accessor(runtimeObj)
	if err != nil {
		return err
	}

	// do not try to add finalizer if deletionTime already exists
	if metaObj.GetDeletionTimestamp() != nil {
		return nil
	}

	currentFinalizers := metaObj.GetFinalizers()
	if _, ok := ListContainsString(finalizer, currentFinalizers); !ok {
		currentFinalizers = append(currentFinalizers, finalizer)
		metaObj.SetFinalizers(currentFinalizers)
		return client.Update(context.TODO(), runtimeObj)
	}
	return nil
}

func RemoveFinalizer(finalizer string, object runtime.Object, client client.Client) error {
	metaObj, err := meta.Accessor(object)
	if err != nil {
		return err
	}
	currentFinalizers := metaObj.GetFinalizers()
	if idxToRemove, ok := ListContainsString(finalizer, currentFinalizers); ok {
		finalFinalizers := append(currentFinalizers[:idxToRemove], currentFinalizers[idxToRemove+1:]...)
		metaObj.SetFinalizers(finalFinalizers)
		return client.Update(context.TODO(), object)
	}

	return nil
}

func ListContainsString(lookup string, list []string) (int, bool) {
	for idx, ele := range list {
		if ele == lookup {
			return idx, true
		}
	}
	return -1, false
}

func updateCrPhase(phase v1alpha1.Phase, client client.Client, fp *v1alpha1.FargateProfile) error {

	// do not try to update if fp has a deletion timestamp
	if fp.GetDeletionTimestamp() != nil {
		return nil
	}

	if fp.Status.Phase != phase {
		fp.Status.Phase = phase
		return client.Status().Update(context.TODO(), fp)
	}

	return nil
}
