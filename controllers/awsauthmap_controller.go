/*
Copyright 2021.

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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awsauthv1alpha1 "github.com/inovex/aws-auth-controller/api/v1alpha1"
)

// AwsAuthMapReconciler reconciles a AwsAuthMap object
type AwsAuthMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsAuthMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AwsAuthMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	currentVersion, err := r.findConfigMapVersion(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Read Version", "version", currentVersion)

	mapList := &awsauthv1alpha1.AwsAuthMapList{}
	err = r.List(ctx, mapList)

	totalCount := len(mapList.Items)
	matchedCount := 0
	for i, snippet := range mapList.Items {
		matched := snippet.Status.MapVersion == currentVersion
		if matched {
			matchedCount++
		} else {
			snippet.Status.MapVersion = currentVersion
			err = r.Status().Update(ctx, &snippet)
			if err != nil {
				logger.Error(err, "Status Update failed", "name", snippet.Name)
			}
		}
		fmt.Printf("%d: %s  %d(%v)\n", i+1, snippet.Name, snippet.Status.MapVersion, matched)
	}
	fmt.Printf("Matched: %d /%d\n", matchedCount, totalCount)
	/*		intVersion++
			authCM.Data["mapRoles"] = fmt.Sprintf("data version %d", intVersion)
			authCM.ObjectMeta.Annotations["awsauth.io/authversion"] = fmt.Sprintf("%d", intVersion)

			err = r.Update(ctx, authCM)
			if err != nil {
				return ctrl.Result{}, err
			}

			r.Client.*/

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsAuthMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsauthv1alpha1.AwsAuthMap{}).
		Complete(r)
}

func (r *AwsAuthMapReconciler) findConfigMapVersion(ctx context.Context) (int, error) {
	authCM := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: "kube-system",
		Name:      "aws-auth",
	}, authCM)
	if err != nil {
		return 0, err
	}
	version, ok := authCM.ObjectMeta.Annotations["awsauth.io/authversion"]
	if !ok {
		version = "0"
	}
	intVersion, err := strconv.Atoi(version)
	if err != nil {
		return 0, err
	}
	return intVersion, nil
}
