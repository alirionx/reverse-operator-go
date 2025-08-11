package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	reverseproxyv1 "github.com/alirionx/reverse-operator-go/api/v1"
)

const appscapeFinalizer = "app-scape.de/finalizer"

// ReverseProxyEntryReconciler reconciles a ReverseProxyEntry object
type ReverseProxyEntryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=reverse-proxy.app-scape.de,resources=reverseproxyentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=reverse-proxy.app-scape.de,resources=reverseproxyentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=reverse-proxy.app-scape.de,resources=reverseproxyentries/finalizers,verbs=update

func (r *ReverseProxyEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	rpe := &reverseproxyv1.ReverseProxyEntry{}
	if err := r.Get(ctx, req.NamespacedName, rpe); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if rpe.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(rpe, appscapeFinalizer) {
			patch := client.MergeFrom(rpe.DeepCopy())
			controllerutil.AddFinalizer(rpe, appscapeFinalizer)
			if err := r.Patch(ctx, rpe, patch); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(rpe, appscapeFinalizer) {
			err := r.cleanupResources(ctx, rpe)
			if err != nil {
				return ctrl.Result{}, err
			}
			logf.Log.Info("Sub Resources (Service, Endpoints, Ingress) deleted", "name", req.NamespacedName)
			patch := client.MergeFrom(rpe.DeepCopy())
			controllerutil.RemoveFinalizer(rpe, appscapeFinalizer)
			if err := r.Patch(ctx, rpe, patch); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	//------------------------------------
	svc := r.reverseService(rpe)
	foundSvc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.Create(ctx, svc)
			if err != nil {
				logf.Log.Error(err, "unable to create Service")
				return ctrl.Result{}, err
			}
			logf.Log.Info("Service created", "name", req.NamespacedName)
		} else {
			return ctrl.Result{}, err
		}
	}
	//----------
	eps := r.reverseEndpoints(rpe)
	foundEps := &discoveryv1.EndpointSlice{}
	err = r.Get(ctx, client.ObjectKey{Name: eps.Name, Namespace: eps.Namespace}, foundEps)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.Create(ctx, eps)
			if err != nil {
				logf.Log.Error(err, "unable to create Endpoints")
				return ctrl.Result{}, err
			}
			logf.Log.Info("Endpoints created", "name", req.NamespacedName)
		} else {
			return ctrl.Result{}, err
		}
	}
	//----------
	ing := r.reverseIngress(rpe)
	foundIng := &networkingv1.Ingress{}
	err = r.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, foundIng)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.Create(ctx, ing)
			if err != nil {
				logf.Log.Error(err, "unable to create Ingress")
				return ctrl.Result{}, err
			}
			logf.Log.Info("Ingress created", "name", req.NamespacedName)
		} else {
			return ctrl.Result{}, err
		}
	}

	//------------------------------------
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReverseProxyEntryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reverseproxyv1.ReverseProxyEntry{}).
		Owns(&corev1.Service{}).
		Owns(&discoveryv1.EndpointSlice{}).
		Owns(&networkingv1.Ingress{}).
		Named("reverseproxyentry").
		Complete(r)
}
