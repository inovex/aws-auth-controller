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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MapRolesSpec defines a mapping of an IAM role to an RBAC user and to RBAC groups.
type MapRolesSpec struct {
	RoleArn  string   `json:"rolearn"`
	UserName string   `json:"username"`
	Groups   []string `json:"groups"`
}

// MapUsersSpec defines a mapping of an IAM user to an RBAC user and to RBAC groups.
type MapUsersSpec struct {
	UserArn  string   `json:"userarn"`
	UserName string   `json:"username"`
	Groups   []string `json:"groups"`
}

// AwsAuthMapSpec defines the IAM role and user mappings to RBAC.
type AwsAuthMapSpec struct {
	MapRoles []MapRolesSpec `json:"mapRoles,omitempty"`
	MapUsers []MapUsersSpec `json:"mapUsers,omitempty"`
}

// AwsAuthMapStatus defines the observed state of AwsAuthMap.
type AwsAuthMapStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	MapVersion int `json:"mapVersion"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="MapVersion",type=integer,JSONPath=`.status.mapVersion`

// AwsAuthMap is the Schema for the awsauthmaps API
type AwsAuthMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsAuthMapSpec   `json:"spec,omitempty"`
	Status AwsAuthMapStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// AwsAuthMapList contains a list of AwsAuthMap
type AwsAuthMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsAuthMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsAuthMap{}, &AwsAuthMapList{})
}
