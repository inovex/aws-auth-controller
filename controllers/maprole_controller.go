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

// MapRoleReconciler reconciles a MapRole object
type MapRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=aws-auth.inovex.de,resources=maproles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws-auth.inovex.de,resources=maproles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws-auth.inovex.de,resources=maproles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MapRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MapRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	authCM := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: "kube-system",
		Name:      "aws-auth",
	}, authCM)
	if err != nil {
		return ctrl.Result{}, err
	}
	version, ok := authCM.ObjectMeta.Annotations["aws-auth.inovex.de/authversion"]
	if !ok {
		version = "0"
	}
	intVersion, _ := strconv.Atoi(version)

	fmt.Printf("Raw Data: %v\n", authCM.Data)

	intVersion++
	authCM.Data["mapRoles"] = fmt.Sprintf("data version %d", intVersion)
	authCM.ObjectMeta.Annotations["aws-auth.inovex.de/authversion"] = fmt.Sprintf("%d", intVersion)

	err = r.Update(ctx, authCM)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MapRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsauthv1alpha1.MapRole{}).
		Complete(r)
}
