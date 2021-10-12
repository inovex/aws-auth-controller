package controllers

import (
	"context"

	crdv1beta1 "github.com/inovex/aws-auth-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const CONFIG_MAP_NAMESPACE = "kube-system"
const CONFIG_MAP_NAME = "aws-auth"
const MAP_ROLES_KEY = "mapRoles"
const MAP_USERS_KEY = "mapUsers"
const MANAGED_ANNOTATION = "awsauth.io/managed"

type MapRoles []crdv1beta1.MapRolesSpec
type MapUsers []crdv1beta1.MapUsersSpec

type MapRolesByArn map[string]crdv1beta1.MapRolesSpec
type MapUsersByArn map[string]crdv1beta1.MapUsersSpec

type AwsAuthMap struct {
	client.Client
	ConfigMap *corev1.ConfigMap
	Roles     *MapRolesByArn
	Users     *MapUsersByArn
}

/*
GetAwsAuthMap constructs an AwsAuthMap instance and fills it with th data from
the ConfigMap.

If the ConfigMap did not yet exist it is created,
*/
func GetAwsAuthMap(client client.Client, ctx context.Context) (*AwsAuthMap, error) {
	awsauthmap := &AwsAuthMap{Client: client}
	if err := awsauthmap.Read(ctx); err != nil {
		return nil, err
	}
	return awsauthmap, nil
}

/*
getOrCreate reads the ConfigMap from the API or creates it if it did not exist.
*/
func (a *AwsAuthMap) getOrCreate(ctx context.Context) error {
	authCM := &corev1.ConfigMap{}
	err := a.Get(ctx, client.ObjectKey{
		Namespace: CONFIG_MAP_NAMESPACE,
		Name:      CONFIG_MAP_NAME,
	}, authCM)

	if err != nil {
		// Check for missing ConfigMap and create
		if apierrs.IsNotFound(err) {
			authCM.ObjectMeta.Namespace = CONFIG_MAP_NAMESPACE
			authCM.ObjectMeta.Name = CONFIG_MAP_NAME
			authCM.Data = make(map[string]string)
			authCM.Data[MAP_ROLES_KEY] = ""
			authCM.Data[MAP_USERS_KEY] = ""
			err = a.Create(ctx, authCM)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	a.ConfigMap = authCM
	return nil
}

/*
Read retrieves the ConfigMap, deserializes the contained YaML objects and maps
their content by ARN for easy manipulation.
*/
func (a *AwsAuthMap) Read(ctx context.Context) error {
	rolesByArn := MapRolesByArn{}
	usersByArn := MapUsersByArn{}

	err := a.getOrCreate(ctx)
	if err != nil {
		return err
	}
	// Create map of MapRolesSpec
	currentMapRoles := &MapRoles{}
	err = yaml.Unmarshal([]byte(a.ConfigMap.Data[MAP_ROLES_KEY]), currentMapRoles)
	if err != nil {
		return err
	}
	for _, cmr := range *currentMapRoles {
		rolesByArn[cmr.RoleArn] = cmr
	}
	// Create map of MapUsersSpec
	currentMapUsers := &MapUsers{}
	err = yaml.Unmarshal([]byte(a.ConfigMap.Data[MAP_USERS_KEY]), currentMapUsers)
	if err != nil {
		return err
	}
	for _, cmu := range *currentMapUsers {
		usersByArn[cmu.UserArn] = cmu
	}
	a.Roles = &rolesByArn
	a.Users = &usersByArn
	return nil
}

/*
Write serializes the mappings back to YaM and writes them to the ConfigMap
API object.

It is important to re-use the ConfigMap object that was retrieved by Read so
that the contained ResourceVersion attribute can be evaluated by the API server
to catch concurrent writes.
*/
func (a *AwsAuthMap) Write(ctx context.Context) error {

	mapRoles := MapRoles{}
	mapUsers := MapUsers{}

	// Create arrays of map content
	for _, rba := range *a.Roles {
		mapRoles = append(mapRoles, rba)
	}
	for _, uba := range *a.Users {
		mapUsers = append(mapUsers, uba)
	}

	// Make Yaml strings from arrays
	mapRolesYaml, err := yaml.Marshal(mapRoles)
	if err != nil {
		return err
	}
	mapUsersYaml, err := yaml.Marshal(mapUsers)
	if err != nil {
		return err
	}

	if a.ConfigMap.ObjectMeta.Annotations == nil {
		// No annotations yet.
		a.ConfigMap.ObjectMeta.Annotations = make(map[string]string)
	}
	a.ConfigMap.ObjectMeta.Annotations[MANAGED_ANNOTATION] = "true"

	// Store Yaml mappings in ConfigMap.
	a.ConfigMap.Data[MAP_ROLES_KEY] = string(mapRolesYaml)
	a.ConfigMap.Data[MAP_USERS_KEY] = string(mapUsersYaml)

	err = a.Update(ctx, a.ConfigMap)
	if err != nil {
		// TODO: Deal with 409 responses (concurrent write).
		return err
	}

	return nil
}
