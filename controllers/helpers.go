package controllers

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"strings"
)

func eksClusterExists(eksClient eksiface.EKSAPI, clusterName string) (*eks.DescribeClusterOutput, bool, error) {

	in := &eks.DescribeClusterInput{Name: aws.String(clusterName)}
	out, err := eksClient.DescribeCluster(in)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == eks.ErrCodeResourceNotFoundException {
				return nil, false, nil
			}
		}
		return nil, false, err
	}

	return out, true, nil
}

func subnetCheck(subnetsToCheck []string, vpcID string, ec2Client ec2iface.EC2API) error {

	// list route tables of that vpc for the N subnets associations
	// aws will not complain if one of the subnet association does not exist at all
	out, err := ec2Client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcID}),
			},
			{
				Name:   aws.String("association.subnet-id"),
				Values: aws.StringSlice(subnetsToCheck),
			},
		},
	})
	if err != nil {
		return err
	}

	subnetsFoundInAws := routeTablesToSubnetIDMap(out.RouteTables)
	for _, subnetID := range subnetsToCheck {
		// ensure subnetID is valid and is within the same cluster VPC
		routes, subnetFoundInAws := subnetsFoundInAws[subnetID]
		if !subnetFoundInAws {
			return ErrInvalidSubnet{Message: fmt.Sprintf("Subnet %v either does not"+
				" exist or is not associated to any route table", subnetID)}
		}

		if !isSubnetPrivate(routes) {
			return ErrInvalidSubnet{Message: fmt.Sprintf("Subnet %v is not a private subnet."+
				"EKS Fargat subnets must be private", subnetID)}
		}
	}
	return nil
}

func routeTablesToSubnetIDMap(rts []*ec2.RouteTable) map[string][]*ec2.Route {
	subnetsFoundAttached := map[string][]*ec2.Route{}
	for _, rt := range rts {
		for _, rtA := range rt.Associations {
			subnetsFoundAttached[*rtA.SubnetId] = rt.Routes
		}
	}
	return subnetsFoundAttached
}

func isSubnetPrivate(r []*ec2.Route) bool {
	for _, rt := range r {
		if rt.DestinationCidrBlock != nil && rt.GatewayId != nil {
			if *rt.DestinationCidrBlock == "0.0.0.0/0" && strings.HasPrefix(*rt.GatewayId, "igw-") {
				return false
			}
		}
	}
	return true
}
