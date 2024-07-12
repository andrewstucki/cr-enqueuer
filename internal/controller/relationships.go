package controller

import (
	"context"

	relationv1alpha1 "github.com/andrewstucki/cr-enqueuer/api/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	childParentIndex = "__child_referencing_parent"
)

func childParent(child *relationv1alpha1.Child) types.NamespacedName {
	return types.NamespacedName{Namespace: child.Namespace, Name: child.Spec.Parent}
}

func registerChildParentIndex(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &relationv1alpha1.Child{}, childParentIndex, indexChildParent)
}

func indexChildParent(o client.Object) []string {
	child := o.(*relationv1alpha1.Child)
	return []string{childParent(child).String()}
}

func childrenForParent(ctx context.Context, c client.Client, nn types.NamespacedName) ([]reconcile.Request, error) {
	childList := &relationv1alpha1.ChildList{}
	err := c.List(ctx, childList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(childParentIndex, nn.String()),
	})

	if err != nil {
		return nil, err
	}

	requests := []reconcile.Request{}
	for _, item := range childList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		})
	}

	return requests, nil
}
