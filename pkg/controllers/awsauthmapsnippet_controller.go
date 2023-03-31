/*
Copyright 2021 inovex GmbH

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

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crdv1beta1 "github.com/inovex/aws-auth-controller/pkg/api/v1beta1"
)

// AwsAuthMapSnippetReconciler reconciles an AwsAuthMapSnippet object
type AwsAuthMapSnippetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const FINALIZER_NAME = "awsauth.io/finalizer"

//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmapsnippets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmapsnippets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.awsauth.io,resources=awsauthmapsnippets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AwsAuthMapSnippetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile Request received", "objectName", req.NamespacedName)

	awsauthmap, err := GetAwsAuthMap(r.Client, ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Read config map", "Roles", awsauthmap.Roles, "Users", awsauthmap.Users)

	snippet := &crdv1beta1.AwsAuthMapSnippet{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      req.NamespacedName.Name,
		Namespace: req.NamespacedName.Namespace},
		snippet)

	if err != nil {
		if apierrs.IsNotFound(err) {
			logger.Info("Resource already deleted, Reconciliation not needed.")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if snippet.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(snippet.GetFinalizers(), FINALIZER_NAME) {
			controllerutil.AddFinalizer(snippet, FINALIZER_NAME)
			if err := r.Update(ctx, snippet); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(snippet.GetFinalizers(), FINALIZER_NAME) {
			// our finalizer is present, so lets handle any external dependency
			logger.Info("Finalizer called")
			if err := r.CleanUpConfigMap(ctx, snippet, awsauthmap); err != nil {
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(snippet, FINALIZER_NAME)
			if err := r.Update(ctx, snippet); err != nil {
				return ctrl.Result{}, err
			}
		}
		logger.Info("Finalizing completed")
		return ctrl.Result{}, nil
	}

	// Prepare patch, remember original resource content
	original := snippet.DeepCopy()
	defer func() {
		if err := r.UpdateSnippetStatus(ctx, snippet, original); err != nil {
			logger.Error(err, "Failed to update status")
		}
	}()

	snippet.Status.IsSynced = false

	logger.Info("Updating ConfigMap")
	if err := r.UpdateConfigMap(ctx, snippet, awsauthmap); err != nil {
		logger.Error(err, "Failed to update ConfigMap")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciliation completed")
	snippet.Status.IsSynced = true

	return ctrl.Result{}, nil
}

/*
UpdateSnippetStatus stores the ARNS that are being managed in the status
sub-object und updates the status.
*/
func (r *AwsAuthMapSnippetReconciler) UpdateSnippetStatus(ctx context.Context, current, original *crdv1beta1.AwsAuthMapSnippet) error {

	// Overwrite lists with current
	current.Status.RoleArns = []string{}
	current.Status.UserArns = []string{}

	for _, mr := range current.Spec.MapRoles {
		current.Status.RoleArns = append(current.Status.RoleArns, mr.RoleArn)
	}
	for _, mu := range current.Spec.MapUsers {
		current.Status.UserArns = append(current.Status.UserArns, mu.UserArn)
	}
	return r.Status().Patch(ctx, current, client.MergeFrom(original))
}

/*
UpdateConfigMap removes obsolete entries from the ConfigMap and updates all
others.

This also covers creation of new entries.
*/
func (r *AwsAuthMapSnippetReconciler) UpdateConfigMap(ctx context.Context, snippet *crdv1beta1.AwsAuthMapSnippet, awsauth *AwsAuthMap) error {

	// find entries that need to be deleted
	// (present in status, missing in spec)
	for _, ra := range snippet.Status.RoleArns {
		found := false
		for _, role := range snippet.Spec.MapRoles {
			if role.RoleArn == ra {
				found = true
				break
			}
		}
		if !found {
			delete(*awsauth.Roles, ra)
		}
	}
	for _, ua := range snippet.Status.UserArns {
		found := false
		for _, user := range snippet.Spec.MapUsers {
			if user.UserArn == ua {
				found = true
				break
			}
		}
		if !found {
			delete(*awsauth.Users, ua)
		}
	}

	// Update or create ARN mappings in the ConfigMap
	for _, mr := range snippet.Spec.MapRoles {
		(*awsauth.Roles)[mr.RoleArn] = mr
	}
	for _, mu := range snippet.Spec.MapUsers {
		(*awsauth.Users)[mu.UserArn] = mu
	}

	// Write the updated ConfigMap to the API
	err := awsauth.Write(ctx)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error(err, "Error updating aws-auth ConfigMap")
		return err
	}
	return nil
}

/*
CleanUpConfigMap removes all ARN mappings from the ConfigMap that were managed
by this snippet.
*/
func (r *AwsAuthMapSnippetReconciler) CleanUpConfigMap(ctx context.Context, snippet *crdv1beta1.AwsAuthMapSnippet, awsauth *AwsAuthMap) error {
	for _, ra := range snippet.Status.RoleArns {
		delete(*awsauth.Roles, ra)
	}
	for _, ua := range snippet.Status.UserArns {
		delete(*awsauth.Users, ua)
	}

	// Write the updated ConfigMap to the API
	err := awsauth.Write(ctx)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error(err, "Error updating aws-auth ConfigMap")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsAuthMapSnippetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1beta1.AwsAuthMapSnippet{}).
		Complete(r)
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
