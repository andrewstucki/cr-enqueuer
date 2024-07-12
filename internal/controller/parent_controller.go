package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	relationv1alpha1 "github.com/andrewstucki/cr-enqueuer/api/v1alpha1"
)

type ParentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=parents/finalizers,verbs=update

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

func (r *ParentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&relationv1alpha1.Parent{}).
		Watches(&relationv1alpha1.Child{}, EnqueueGenericReference(childParent)).
		Complete(r)
}
