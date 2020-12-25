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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
	ec2Client := NewEc2Client(cr.Spec.Region)
	iamClient := NewIamClient(cr.Spec.Region)

	// add finalizers
	if errAddingFinalizer := AddFinalizer(FargateProfileFinalizer, cr, r.Client); errAddingFinalizer != nil {
		r.Log.Error(errAddingFinalizer, fmt.Sprintf("Failed to add finalizer to %s", req.NamespacedName.String()))
		return ctrl.Result{}, errAddingFinalizer
	}

	// handle delete
	if cr.GetDeletionTimestamp() != nil {

		if errMarkingFpDeleting := updateCrPhase(agillappsv1alpha1.Deleting, r.Client, cr); errMarkingFpDeleting != nil {
			return ctrl.Result{}, errMarkingFpDeleting
		}

		if errDeletingFprofile := deleteFprofile(cr.WithDeleteIn(), eksClient); errDeletingFprofile != nil {
			r.Log.Error(errDeletingFprofile, "Failed to delete fargate-profile")
			return ctrl.Result{}, errDeletingFprofile
		}
		if errRemovingFinalizer := RemoveFinalizer(FargateProfileFinalizer, cr, r.Client); errRemovingFinalizer != nil {
			return ctrl.Result{}, errRemovingFinalizer
		}
		r.Log.Info(fmt.Sprintf("%s: Successfully deleted fargate-profile", req.NamespacedName.String()))
		return ctrl.Result{}, nil
	}

	// run some checks before attempting to create anything
	if errCheckingPreReqs := runPreFlightChecks(eksClient, ec2Client, iamClient, cr); errCheckingPreReqs != nil {
		switch e := errCheckingPreReqs.(type) {

		case ErrEksClusterNotFound:
			r.Log.Error(e, fmt.Sprintf("%v: %v eks cluster "+
				"does not exist", req.NamespacedName, cr.Spec.ClusterName))
			return ctrl.Result{}, updateCrPhase(agillappsv1alpha1.Failed, r.Client, cr)

		case ErrEksClusterNotActive:
			r.Log.Info(fmt.Sprintf("%v: %v eks cluster is not in active state."+
				" Will check back in few mins", req.NamespacedName, cr.Spec.ClusterName))
			return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil

		case ErrPodExecutionRoleArnNotFound:
			r.Log.Info(fmt.Sprintf("%v: %v pod execution role arn does not exist."+
				"Please update spec with correct podExecutionRoleArn", req.NamespacedName, cr.Spec.PodExecutionRoleArn))
			return ctrl.Result{}, updateCrPhase(agillappsv1alpha1.Failed, r.Client, cr)

		case ErrInvalidSubnet:
			r.Log.Error(e, fmt.Sprintf("%v: has invalid subnets - %v. "+
				"Please update spec with correct subnets", req.NamespacedName, e.Message))
			return ctrl.Result{}, updateCrPhase(agillappsv1alpha1.Failed, r.Client, cr)

		default:
			r.Log.Error(e, "Something went wrong while running pre-flight checks")
			return ctrl.Result{}, updateCrPhase(agillappsv1alpha1.Failed, r.Client, cr)
		}
	}

	// describe fProfile
	fpState, errDescribingFp := eksClient.DescribeFargateProfile(&eks.DescribeFargateProfileInput{
		ClusterName:        aws.String(cr.Spec.ClusterName),
		FargateProfileName: aws.String(cr.GetName()),
	})
	if errDescribingFp != nil {

		if awsErr, isAwsErr := errDescribingFp.(awserr.Error); isAwsErr {
			// not found, create it
			if awsErr.Code() == eks.ErrCodeResourceNotFoundException {
				if errCreatingFProfile := createFProfile(cr.WithCreateIn(), eksClient); errCreatingFProfile != nil {
					r.Log.Error(errCreatingFProfile, "Failed to create fargate-profile")
					return ctrl.Result{}, errCreatingFProfile
				}
				r.Log.Info(fmt.Sprintf("%s: Creating fargate-profile", req.NamespacedName.String()))
				return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, updateCrPhase(agillappsv1alpha1.Creating, r.Client, cr)
			}
		}
		// not-recognized error, requeue
		r.Log.Error(errDescribingFp, "Failed to describe fargate-profile")
		return ctrl.Result{}, errDescribingFp
	}

	currentFpStatus := *fpState.FargateProfile.Status
	if currentFpStatus != eks.FargateProfileStatusActive {
		r.Log.Info(fmt.Sprintf("%s: fargate-profile is not active yet. Current status: %v", req.NamespacedName.String(), currentFpStatus))
		return ctrl.Result{RequeueAfter: time.Minute, Requeue: true}, nil
	}
	r.Log.Info(fmt.Sprintf("%v: fargate-profile is %v", req.NamespacedName, currentFpStatus))
	return ctrl.Result{}, updateCrPhase(agillappsv1alpha1.Ready, r.Client, cr)
}

func (r *FargateProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agillappsv1alpha1.FargateProfile{}).
		WithEventFilter(predicate.Funcs{

			// must return true to let this event reconcile
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
			},
		}).
		Complete(r)
}
