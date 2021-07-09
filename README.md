# aws-auth-controller
**Managing the aws-auth ConfigMap**

## Problem

Access to the Kubernetes-API of an EKS cluster is managed through IAM roles
that are mapped to RBAC users and groups. This mapping is stored in a configmap
named `aws-auth` in the `kube-system` namespace.

This configmap is created by EKS as soon as the first nodegroup is created so
that the nodes themselves can access the API. To grant access for other users,
this configmap needs to be edited. The details are found in the
[EKS documentation](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html).

This editing procedure is a bit complex to automate in CI. Due to the
centralized nature of the configmap it is also difficult to add and remove
entries in a distruibuted manner, depending on your deployment strategy.

## The solution

The controller takes care of editing the `aws-auth` configmap. It takes
snippets of it as input and makes sure the configmap stays in sync with
the desired access configuration.

## Usage

The snippets are defined in custom resources that can be placed in any
namespace throughout the cluster. They follow the same semantics as the
`aws-auth` configmap itself.

To pick up on the example from the AWS documentation, that snippet would
look like this in the CR:

    apiVersion: crd.awsauth.io/v1beta1
    kind: AwsAuthMap
    metadata:
      name: awsauthmap-sample
      namespace: sample-namespace
    spec:
      mapRoles:
        - rolearn: arn:aws:iam::111122223333:role/eksctl-my-cluster-nodegroup-standard-wo-NodeInstanceRole-1WP3NUE3O6UCF
          username: system:node:{{EC2PrivateDNSName}}
          groups:
            - system:bootstrappers
            - system:nodes
      mapUsers:
        - userarn: arn:aws:iam::111122223333:user/admin
          username: admin
          groups:
            - system:masters
        - userarn: arn:aws:iam::111122223333:user/ops-user
          username: ops-user
          groups:
            - system:masters

When this resource is added to the cluster, the controller will modify the
configmap to include the entries in this snippet. When it is removed, the
respective entries will be removed, too.

## Implementation details

The controller adds an annotation to the configmap `awsauth.io/authversion`
that holds an incrementing serial number to match the configmap to the
custom resources in the cluster. The same value is stored in the CR's
status so that out-of-sync resources can be identified.

The controller calculates the SHA256 checksum of the snippet's content
(arns, usernames, groupnames) to detect changed snippets that need to be
updated in the configmap. The checksum is also stored in the status.

## Controller deployment

A working single-file deployment manifest is forthcoming. For now the
manifests under configs can be used as a guide.

The CRD is best created this way:

    make manifests
    kubectl kustomize config/crd > crd.yaml

## Development

This project was created with kubebuilder 3.1.8 using go 1.16. See the
[documentation](https://kubebuilder.io/) for more information.

To try out the controller locally against a running cluster, simply do

    make install # apply crd to cluster
    make run ENABLE_WEBHOOKS=false

It will use whatever kubectl context is currently active.

## TODOs

  * Semantic release versioning
  * Clean up repository
  * A validating webhook to
    * check the ARN format
    * check for duplicate ARNs
    * check with AWS if ARN actually exists (?, needs IRSA)
  * Add `mapAccounts` if anybody needs it
  * Tests, automated
