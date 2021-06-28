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
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	awsauthv1alpha1 "github.com/inovex/aws-auth-controller/api/v1alpha1"
)

// AwsAuthMapReconciler reconciles a AwsAuthMap object
type AwsAuthMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type MapRoles []awsauthv1alpha1.MapRolesSpec
type MapUsers []awsauthv1alpha1.MapUsersSpec

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

	logger.Info("Reconcile Request received", "objectName", req.NamespacedName)
	currentVersion, err := r.findConfigMapVersion(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Read Version", "version", currentVersion)

	thisMap := &awsauthv1alpha1.AwsAuthMap{}
	err = r.Get(ctx, client.ObjectKey{
		Name: req.NamespacedName.Name}, thisMap)

	outOfSync := false

	if err == nil {
		if thisMap.IsChanged() {
			logger.Info("Content changed, updating.")
			outOfSync = true
		} else if thisMap.Status.MapVersion == currentVersion {
			logger.Info("Already synced", "version", currentVersion)
			return ctrl.Result{}, nil
		}
	} else if apierrs.IsNotFound(err) {
		logger.Info("Deletion event, updating.")
		outOfSync = true
	} else {
		fmt.Printf("Error: %v\n", err)
		return ctrl.Result{}, err
	}

	mapList := &awsauthv1alpha1.AwsAuthMapList{}
	//selector := fields.ParseSelectorOrDie(fmt.Sprintf("status.mapVersion!=%d", currentVersion))
	err = r.List(ctx, mapList /*, client.MatchingFieldsSelector{selector}*/)

	totalCount := len(mapList.Items)
	mapRoles := MapRoles{}
	mapUsers := MapUsers{}
	for _, authMap := range mapList.Items {
		if authMap.Status.MapVersion != currentVersion {
			outOfSync = true
			logger.Info("Found out-of-sync map", "name", authMap.Name, "version", authMap.Status.MapVersion)
		}
		mapRoles = append(mapRoles, authMap.Spec.MapRoles...)
		mapUsers = append(mapUsers, authMap.Spec.MapUsers...)
	}
	logger.Info("Maps counted", "total", totalCount, "out-of-sync", outOfSync)

	if !outOfSync {
		return ctrl.Result{}, nil
	}

	currentVersion++

	err = r.updateConfigMap(ctx, mapRoles, mapUsers, currentVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, authMap := range mapList.Items {
		authMap.Status.MapVersion = currentVersion
		authMap.Status.CheckSum = authMap.Spec.CalcCheckSum()
		err = r.Status().Update(ctx, &authMap)
		if err != nil {
			logger.Error(err, "Status Update failed", "name", authMap.Name, "version", currentVersion)
		} else {
			logger.Info("Status updated", "name", authMap.Name, "version", currentVersion)
		}
	}
	logger.Info("Reconciliation completed", "newVersion", currentVersion)
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

func (r *AwsAuthMapReconciler) updateConfigMap(ctx context.Context, mapRoles MapRoles, mapUsers MapUsers, version int) error {
	mapRolesYaml, err := yaml.Marshal(mapRoles)
	if err != nil {
		return err
	}
	mapUsersYaml, err := yaml.Marshal(mapUsers)
	if err != nil {
		return err
	}

	authCM := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: "kube-system",
		Name:      "aws-auth",
	}, authCM)
	if err != nil {
		return err
	}

	authCM.ObjectMeta.Annotations["awsauth.io/authversion"] = fmt.Sprintf("%d", version)
	authCM.Data["mapRoles"] = string(mapRolesYaml)
	authCM.Data["mapUsers"] = string(mapUsersYaml)

	err = r.Update(ctx, authCM)
	if err != nil {
		return err
	}

	return nil
}
