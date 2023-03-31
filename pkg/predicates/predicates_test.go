package predicates

import (
	"testing"

	crdv1beta1 "github.com/inovex/aws-auth-controller/pkg/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

func TestPredicates(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Predicates Suite")
}

var _ = Context("Predicates", func() {
	Describe("namespaceFilter", func() {
		DescribeTable("filter for namespaces", func(namespace string, watched []string, result bool) {
			obj := &crdv1beta1.AwsAuthMapSnippet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-bar",
					Namespace: namespace,
				},
			}
			pred := NamespaceFilter(watched)
			Expect(pred.Create(event.CreateEvent{Object: obj})).To(Equal(result))
		},
			Entry("with no filter", "anyns", []string{}, true),
			Entry("with empty filter", "anyns", []string{""}, true),
			Entry("with single filter", "myns", []string{"myns"}, true),
			Entry("with single filter skipped", "anyns", []string{"myns"}, false),
			Entry("with multiple filter", "myns", []string{"myns", "otherns"}, true),
			Entry("with multple filter skipped", "anyns", []string{"myns", "otherns"}, false),
		)
	})
})
