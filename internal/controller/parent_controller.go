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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	relationv1alpha1 "github.com/andrewstucki/cr-enqueuer/api/v1alpha1"
)

// ParentReconciler reconciles a Parent object
type ParentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Parent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ParentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling parent", "parent", req.NamespacedName)

	parent := &relationv1alpha1.Parent{}
	if err := r.Get(ctx, req.NamespacedName, parent); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	children, err := childrenForParent(ctx, r.Client, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	if parent.Status.Children == len(children) {
		return ctrl.Result{}, nil
	}

	parent.Status.Children = len(children)

	return ctrl.Result{}, r.Status().Update(ctx, parent)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ParentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&relationv1alpha1.Parent{}).
		Watches(&relationv1alpha1.Child{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: childParent(o.(*relationv1alpha1.Child))}}
		})).
		Complete(r)
}
