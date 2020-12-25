/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Phase string

const (
	Ready             Phase = "Ready"
	Creating          Phase = "Creating"
	Deleting          Phase = "Deleting"
	Failed            Phase = "Failed"
	PendingEvaluation Phase = "Pending-Evaluation"
)

type FargateProfileSelector struct {
	// The Kubernetes labels that the selector should match. A pod must contain
	// all of the labels that are specified in the selector for it to be considered
	// a match.
	Labels    map[string]string `json:"labels"`
	Namespace string            `json:"namespace,required"`
}

// FargateProfileSpec defines the desired state of FargateProfile
type FargateProfileSpec struct {
	Region string `json:"region,required"`

	// The name of the Amazon EKS cluster to apply the Fargate profile to.
	// ClusterName is a required field
	ClusterName string `json:"clusterName,required"`

	// The Amazon Resource Name (ARN) of the pod execution role to use for pods
	// that match the selectors in the Fargate profile. The pod execution role allows
	// Fargate infrastructure to register with your cluster as a node, and it provides
	// read access to Amazon ECR image repositories. For more information, see Pod
	// Execution Role (https://docs.aws.amazon.com/eks/latest/userguide/pod-execution-role.html)
	// in the Amazon EKS User Guide.
	// PodExecutionRoleArn is a required field
	PodExecutionRoleArn string `json:"podExecutionRoleArn,required"`

	// An object representing an AWS Fargate profile selector ( can include 5 at max ).
	// +kubebuilder:validation:MaxItems=5
	// +kubebuilder:validation:MinItems=1
	Selectors []FargateProfileSelector `json:"selectors"`

	// The IDs of subnets to launch your pods into. At this time, pods running on
	// Fargate are not assigned public IP addresses, so only private subnets (with
	// no direct route to an Internet Gateway) are accepted for this parameter.
	// TODO: add validation in each reconcile to make sure subnets provided are private
	Subnets []string `json:"subnets"`

	// The metadata to apply to the Fargate profile to assist with categorization
	// and organization. Each tag consists of a key and an optional value, both
	// of which you define. Fargate profile tags do not propagate to any other resources
	// associated with the Fargate profile, such as the pods that are scheduled
	// with it.
	// +optional
	Tags map[string]string `json:"tags"`
}

// FargateProfileStatus defines the observed state of FargateProfile
type FargateProfileStatus struct {
	Phase Phase `json:"phase"`
}

// +kubebuilder:object:root=true

// FargateProfile is the Schema for the fargateprofiles API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="selectors",type=string,JSONPath=`.spec.selectors`
// +kubebuilder:printcolumn:name="phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type FargateProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FargateProfileSpec   `json:"spec,omitempty"`
	Status FargateProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FargateProfileList contains a list of FargateProfile
type FargateProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FargateProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FargateProfile{}, &FargateProfileList{})
}

func (in *FargateProfile) WithCreateIn() *eks.CreateFargateProfileInput {

	selectorsFn := func() []*eks.FargateProfileSelector {
		var s []*eks.FargateProfileSelector
		for _, inS := range in.Spec.Selectors {
			s = append(s, &eks.FargateProfileSelector{
				Labels:    aws.StringMap(inS.Labels),
				Namespace: aws.String(inS.Namespace),
			})
		}
		return s
	}

	out := &eks.CreateFargateProfileInput{
		ClusterName:         aws.String(in.Spec.ClusterName),
		FargateProfileName:  aws.String(in.GetName()),
		PodExecutionRoleArn: aws.String(in.Spec.PodExecutionRoleArn),
		Selectors:           selectorsFn(),
		Subnets:             aws.StringSlice(in.Spec.Subnets),
		Tags:                aws.StringMap(in.Spec.Tags),
	}

	return out
}

func (in *FargateProfile) WithDeleteIn() *eks.DeleteFargateProfileInput {
	return &eks.DeleteFargateProfileInput{
		ClusterName:        aws.String(in.Spec.ClusterName),
		FargateProfileName: aws.String(in.GetName()),
	}
}
