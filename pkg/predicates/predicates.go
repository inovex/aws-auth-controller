package predicates

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func NamespaceFilter(namespaces []string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		// No filter specified
		if len(namespaces) == 0 || len(namespaces) == 1 && namespaces[0] == "" {
			return true
		}

		for _, ns := range namespaces {
			if ns == object.GetNamespace() {
				return true
			}
		}
		return false
	})
}
