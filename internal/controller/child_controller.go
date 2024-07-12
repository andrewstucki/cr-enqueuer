/*
Copyright 2024.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	relationv1alpha1 "github.com/andrewstucki/cr-enqueuer/api/v1alpha1"
)

// ChildReconciler reconciles a Child object
type ChildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func childParent(child *relationv1alpha1.Child) types.NamespacedName {
	return types.NamespacedName{Namespace: child.Namespace, Name: child.Spec.Parent}
}

// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Child object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ChildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling child", "child", req.NamespacedName)

	child := &relationv1alpha1.Child{}
	if err := r.Get(ctx, req.NamespacedName, child); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	parent := &relationv1alpha1.Parent{}
	hasParent := true
	if err := r.Get(ctx, childParent(child), parent); err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		hasParent = false
	}

	if child.Status.Bound == hasParent {
		return ctrl.Result{}, nil
	}

	child.Status.Bound = hasParent

	return ctrl.Result{}, r.Status().Update(ctx, child)
}

const (
	Child_Parent = "__child_referencing_parent"
)

// SetupWithManager sets up the controller with the Manager.
func (r *ChildReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &relationv1alpha1.Child{}, Child_Parent, r.indexChildParent); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&relationv1alpha1.Child{}).
		Watches(&relationv1alpha1.Parent{}, EnqueueLossyRequestsFromMapFunc(ctx, 5, r.mapParentToChildren)).
		Complete(r)
}

func (r *ChildReconciler) indexChildParent(o client.Object) []string {
	child := o.(*relationv1alpha1.Child)
	return []string{childParent(child).String()}
}

func (r *ChildReconciler) mapParentToChildren(ctx context.Context, nn types.NamespacedName) ([]reconcile.Request, error) {
	return childrenForParent(ctx, r.Client, nn)
}

func childrenForParent(ctx context.Context, c client.Client, nn types.NamespacedName) ([]reconcile.Request, error) {
	childList := &relationv1alpha1.ChildList{}
	err := c.List(ctx, childList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(Child_Parent, nn.String()),
	})
	if err != nil {
		return nil, err
	}

	children := []reconcile.Request{}
	for _, child := range childList.Items {
		children = append(children, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&child)})
	}

	return children, nil
}
