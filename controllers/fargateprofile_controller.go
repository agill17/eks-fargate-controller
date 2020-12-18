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

package controllers

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agillappsv1alpha1 "github.com/agill17/eks-fargate-controller/api/v1alpha1"
)

// FargateProfileReconciler reconciles a FargateProfile object
type FargateProfileReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=agill.apps.eks-fargate-controller,resources=fargateprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agill.apps.eks-fargate-controller,resources=fargateprofiles/status,verbs=get;update;patch

func (r *FargateProfileReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("fargateprofile", req.NamespacedName)

	cr := &agillappsv1alpha1.FargateProfile{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			// do not requeue
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, fmt.Sprintf("Failed to get CR from %v request", req.Namespace))
		return ctrl.Result{}, err
	}
	eksClient := NewEksClient(cr.Spec.Region)

	// add finalizers
	if errAddingFinalizer := AddFinalizer(FargateProfileFinalizer, cr, r.Client); errAddingFinalizer != nil {
		r.Log.Error(errAddingFinalizer, fmt.Sprintf("Failed to add finalizer to %s", req.NamespacedName.String()))
		return ctrl.Result{}, errAddingFinalizer
	}

	// handle delete
	if cr.GetDeletionTimestamp() != nil {
		deleteIn := &eks.DeleteFargateProfileInput{
			ClusterName:        aws.String(cr.Spec.ClusterName),
			FargateProfileName: aws.String(cr.GetName()),
		}
		if _, errDeleting := eksClient.DeleteFargateProfile(deleteIn); errDeleting != nil {
			if awsErr, ok := errDeleting.(awserr.Error); ok && awsErr.Code() == eks.ErrCodeResourceNotFoundException {
				r.Log.Info(fmt.Sprintf("%v: attempted to delete, but fargate-profile does not exist.. skipping", req.NamespacedName.String()))
				return ctrl.Result{}, RemoveFinalizer(FargateProfileFinalizer, cr, r.Client)
			}
			r.Log.Error(errDeleting, fmt.Sprintf("Could not delete %v fargate profile", req.NamespacedName.String()))
			return ctrl.Result{}, errDeleting
		}

		return ctrl.Result{}, RemoveFinalizer(FargateProfileFinalizer, cr, r.Client)
	}

	// ensure specified eks cluster exists
	eksClusterState, errDescribingCluster := eksClient.DescribeCluster(&eks.DescribeClusterInput{Name: aws.String(cr.Spec.ClusterName)})
	if errDescribingCluster != nil {
		if awsErr, ok := errDescribingCluster.(awserr.Error); ok && awsErr.Code() == eks.ErrCodeResourceNotFoundException {
			r.Log.Error(awsErr, fmt.Sprintf("%v eks cluster not found in %v region", cr.Spec.ClusterName, cr.Spec.Region))
			// delayed retry
			return ctrl.Result{RequeueAfter: 2 * time.Minute}, awsErr
		}
		r.Log.Error(errDescribingCluster, "Could not query the state of EKS cluster")
		return ctrl.Result{}, errDescribingCluster
	}

	// eks cluster is in available state -- TODO: is this necessary or can I narrow down this check more?
	if *eksClusterState.Cluster.Status != eks.ClusterStatusActive {
		r.Log.Info(fmt.Sprintf("%v eks cluster is in %v state, waiting to become active first", cr.Spec.ClusterName, *eksClusterState.Cluster.Status))
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// create fargate profile if not exists
	_, errDescribingFp := eksClient.DescribeFargateProfile(&eks.DescribeFargateProfileInput{
		ClusterName:        aws.String(cr.Spec.ClusterName),
		FargateProfileName: aws.String(cr.GetName()),
	})
	if errDescribingFp != nil {
		if awsErr, isAwsErr := errDescribingFp.(awserr.Error); isAwsErr && awsErr.Code() == eks.ErrCodeResourceNotFoundException {

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
			r.Log.Info(fmt.Sprintf("%v: creating profile", req.NamespacedName.String()))

			if _, errCreatingFargateProfile := eksClient.CreateFargateProfile(createInput); errCreatingFargateProfile != nil {
				r.Log.Error(errCreatingFargateProfile, "Failed to create fargate profile")
				return ctrl.Result{}, errCreatingFargateProfile
			}
			return ctrl.Result{RequeueAfter: time.Minute, Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *FargateProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agillappsv1alpha1.FargateProfile{}).
		Complete(r)
}
