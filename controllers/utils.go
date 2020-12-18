package controllers

import (
	"context"
	"github.com/agill17/eks-fargate-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func UpdateFpCrStatus(status string, fp *v1alpha1.FargateProfile, client client.Client) error {
	if fp.GetDeletionTimestamp() != nil && fp.Status.Phase != status {
		fp.Status.Phase = status
		return client.Status().Update(context.TODO(), fp)
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
