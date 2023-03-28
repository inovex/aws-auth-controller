package controllers

import (
	"context"
	"time"

	crdv1beta1 "github.com/inovex/aws-auth-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

var _ = Describe("snippet controller", func() {
	It("should update status", func() {
		const USER_ARN = "arn:aws:iam::123456789012:user/foobar"
		snip := &crdv1beta1.AwsAuthMapSnippet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsnip",
				Namespace: "default",
			},
			Spec: crdv1beta1.AwsAuthMapSnippetSpec{
				MapUsers: []crdv1beta1.MapUsersSpec{
					{
						UserArn:  USER_ARN,
						UserName: "foobar-name",
						Groups:   []string{"foobar-group"},
					},
				},
			},
			Status: crdv1beta1.AwsAuthMapSnippetStatus{IsSynced: false},
		}
		err := k8sClient.Create(context.Background(), snip)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			// check if status exists
			err = k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      snip.Name,
				Namespace: snip.Namespace,
			}, snip)
			if err != nil {
				return false
			}
			if !snip.Status.IsSynced {
				return false
			}
			if len(snip.Status.UserArns) != 1 {
				return false
			}
			return snip.Status.UserArns[0] == USER_ARN
		}, time.Second*10, time.Second).Should(BeTrue())

	})

	It("should set isSync to false on failure", func() {
		const USER_ARN = "arn:aws:iam::123456789012:user/foobar"
		snip := &crdv1beta1.AwsAuthMapSnippet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsnip2",
				Namespace: "default",
			},
			Spec: crdv1beta1.AwsAuthMapSnippetSpec{
				MapUsers: []crdv1beta1.MapUsersSpec{
					{
						UserArn:  USER_ARN,
						UserName: "foobar-name",
						Groups:   []string{"foobar-group"},
					},
				},
			},
			Status: crdv1beta1.AwsAuthMapSnippetStatus{IsSynced: false},
		}
		k8sClient.FailUpdateName = "aws-auth"
		defer func() {
			k8sClient.FailUpdateName = ""
		}()

		err := k8sClient.Create(context.Background(), snip)
		Expect(err).ToNot(HaveOccurred())

		Consistently(func() bool {
			// check if status exists
			err = k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      snip.Name,
				Namespace: snip.Namespace,
			}, snip)
			if err != nil {
				return true
			}
			return snip.Status.IsSynced
		}, time.Second*10, time.Second).Should(BeFalse())

	})

	It("should create the ConfigMap", func() {
		const USER_ARN = "arn:aws:iam::123456789012:user/foobar"
		snip := &crdv1beta1.AwsAuthMapSnippet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsnip3",
				Namespace: "default",
			},
			Spec: crdv1beta1.AwsAuthMapSnippetSpec{
				MapUsers: []crdv1beta1.MapUsersSpec{
					{
						UserArn:  USER_ARN,
						UserName: "foobar-name",
						Groups:   []string{"foobar-group"},
					},
				},
			},
			Status: crdv1beta1.AwsAuthMapSnippetStatus{IsSynced: false},
		}
		err := k8sClient.Create(context.Background(), snip)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			cm := &corev1.ConfigMap{}
			err = k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      CONFIG_MAP_NAME,
				Namespace: CONFIG_MAP_NAMESPACE,
			}, cm)
			if err != nil {
				return false
			}
			// Check if the mapping exists
			mapYaml, ok := cm.Data[MAP_USERS_KEY]
			if !ok {
				return false
			}
			mapUsers := &MapUsers{}
			// Check if the mapping is correct yaml
			err = yaml.Unmarshal([]byte(mapYaml), mapUsers)
			if err != nil {
				return false
			}
			// Check if the mapping contains the ARN
			for _, user := range *mapUsers {
				if user.UserArn == USER_ARN {
					return true
				}
			}
			return false
		}, time.Second*10, time.Second).Should(BeTrue())
	})
})
