package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type empty struct{}

type LossyMapFunc func(context.Context, types.NamespacedName) ([]reconcile.Request, error)

func EnqueueLossyRequestsFromMapFunc(ctx context.Context, maxRetries int, fn LossyMapFunc) handler.EventHandler {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	go func() {
		<-ctx.Done()
		queue.ShutDownWithDrain()
	}()

	mapper := &lossyEnqueueRequestsFromMapFunc{
		maxRetries: maxRetries,
		toRequests: fn,
		queue:      queue,
	}

	go func() {
		for mapper.processNextItem() {
		}
	}()

	return mapper
}

var _ handler.EventHandler = &lossyEnqueueRequestsFromMapFunc{}

type lossyEnqueueRequestsFromMapFunc struct {
	maxRetries int
	toRequests LossyMapFunc
	queue      workqueue.RateLimitingInterface
}

type enqueuedOp struct {
	ctx   context.Context
	obj   types.NamespacedName
	queue workqueue.RateLimitingInterface
}

func createOp(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface) *enqueuedOp {
	return &enqueuedOp{
		ctx:   ctx,
		obj:   client.ObjectKeyFromObject(obj),
		queue: q,
	}
}

func (e *lossyEnqueueRequestsFromMapFunc) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.queue.Add(createOp(ctx, evt.Object, q))
}

func (e *lossyEnqueueRequestsFromMapFunc) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.queue.Add(createOp(ctx, evt.ObjectOld, q))
	e.queue.Add(createOp(ctx, evt.ObjectNew, q))
}

func (e *lossyEnqueueRequestsFromMapFunc) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.queue.Add(createOp(ctx, evt.Object, q))
}

func (e *lossyEnqueueRequestsFromMapFunc) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.queue.Add(createOp(ctx, evt.Object, q))
}

func (e *lossyEnqueueRequestsFromMapFunc) processNextItem() bool {
	item, quit := e.queue.Get()
	if quit {
		return false
	}

	defer e.queue.Done(item)

	reqs := map[reconcile.Request]empty{}

	op := item.(*enqueuedOp)
	mapped, err := e.toRequests(op.ctx, op.obj)
	if err == nil {
		for _, req := range mapped {
			_, ok := reqs[req]
			if !ok {
				op.queue.Add(req)
				reqs[req] = empty{}
			}
		}
		e.queue.Forget(item)
		return true
	}

	if e.maxRetries != 0 && e.queue.NumRequeues(item) < e.maxRetries {
		// retry instead of erroring
		e.queue.AddRateLimited(item)
		return true
	}

	// we have an error to report
	e.queue.Forget(item)
	runtime.HandleError(err)

	return true
}
