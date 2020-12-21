package controllers

import (
	"fmt"
	"github.com/agill17/eks-fargate-controller/api/v1alpha1"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
)

func runPreFlightChecks(eksClient eksiface.EKSAPI, ec2Client ec2iface.EC2API, cr *v1alpha1.FargateProfile) error {

	// TODO: check if roleArn is valid

	clusterState, clusterExists, errDescribingCluster := eksClusterExists(eksClient, cr.Spec.ClusterName)
	if errDescribingCluster != nil {
		return errDescribingCluster
	}
	if !clusterExists {
		return ErrEksClusterNotFound{Message: fmt.Sprintf("%v eks cluster not found", cr.Spec.ClusterName)}
	}
	if *clusterState.Cluster.Status != eks.ClusterStatusActive {
		return ErrEksClusterNotActive{Message: fmt.Sprintf("%v eks cluster is not yet active", cr.Spec.ClusterName)}
	}

	return subnetCheck(cr.Spec.Subnets, *clusterState.Cluster.ResourcesVpcConfig.VpcId, ec2Client)
}
