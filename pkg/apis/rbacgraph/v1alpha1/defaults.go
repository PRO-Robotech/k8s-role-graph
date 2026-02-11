package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&RoleGraphReview{}, func(obj interface{}) {
		SetObjectDefaults_RoleGraphReview(obj.(*RoleGraphReview))
	})
	return nil
}

func SetObjectDefaults_RoleGraphReview(in *RoleGraphReview) {
	in.EnsureDefaults()
}
