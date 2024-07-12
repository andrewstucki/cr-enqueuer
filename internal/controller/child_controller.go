package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	relationv1alpha1 "github.com/andrewstucki/cr-enqueuer/api/v1alpha1"
)

type ChildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=relation.lambda.coffee,resources=children/finalizers,verbs=update

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

func (r *ChildReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := registerChildParentIndex(ctx, mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&relationv1alpha1.Child{}).
		Watches(&relationv1alpha1.Parent{}, EnqueueLossyRequestsFromMapFunc(ctx, 5, func(ctx context.Context, nn types.NamespacedName) ([]reconcile.Request, error) {
			return childrenForParent(ctx, r.Client, nn)
		})).
		Complete(r)
}
