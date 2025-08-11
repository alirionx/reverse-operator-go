package controller

import (
	"context"
	"fmt"

	reverseproxyv1 "github.com/alirionx/reverse-operator-go/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/util/intstr"
)

var subResourcePrefix string = "rpe-"

func (r *ReverseProxyEntryReconciler) reverseService(rpe *reverseproxyv1.ReverseProxyEntry) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subResourcePrefix + rpe.Name,
			Namespace: rpe.Namespace,
			Labels: map[string]string{
				"app": "reverse-proxy",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(rpe.Spec.Target.Port),
					TargetPort: intstr.FromInt(int(rpe.Spec.Target.Port)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	controllerutil.SetControllerReference(rpe, svc, r.Scheme)
	return svc
}

func (r *ReverseProxyEntryReconciler) reverseEndpoints(rpe *reverseproxyv1.ReverseProxyEntry) *discoveryv1.EndpointSlice {
	eps := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subResourcePrefix + rpe.Name,
			Namespace: rpe.Namespace,
			Labels: map[string]string{
				"app":                        "reverse-proxy",
				"kubernetes.io/service-name": subResourcePrefix + rpe.Name,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{rpe.Spec.Target.Endpoints[0]},
				Conditions: discoveryv1.EndpointConditions{
					Ready: func(b bool) *bool { return &b }(true),
				},
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name:     &rpe.Name,
				Port:     func(p int32) *int32 { return &p }(int32(rpe.Spec.Target.Port)),
				Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
			},
		},
	}
	controllerutil.SetControllerReference(rpe, eps, r.Scheme)
	return eps
}

func (r *ReverseProxyEntryReconciler) reverseIngress(rpe *reverseproxyv1.ReverseProxyEntry) *networkingv1.Ingress {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subResourcePrefix + rpe.Name,
			Namespace: rpe.Namespace,
			Labels: map[string]string{
				"app": "reverse-proxy",
			},
			Annotations: rpe.Spec.Ingress.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &rpe.Spec.Ingress.ClassName,
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						*rpe.Spec.Ingress.Host,
					},
					SecretName: subResourcePrefix + rpe.Name,
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: *rpe.Spec.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: func(p networkingv1.PathType) *networkingv1.PathType { return &p }(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: subResourcePrefix + rpe.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(rpe.Spec.Target.Port),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	controllerutil.SetControllerReference(rpe, ing, r.Scheme)
	return ing
}

// ------------------------------------------------------------
func (r *ReverseProxyEntryReconciler) cleanupResources(ctx context.Context, rpe *reverseproxyv1.ReverseProxyEntry) error {
	var allErrs []error
	var delErr error
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: subResourcePrefix + rpe.Name, Namespace: rpe.Namespace}, svc)
	if err == nil {
		if delErr = r.Delete(ctx, svc); delErr != nil && !apierrors.IsNotFound(err) {
			allErrs = append(allErrs, fmt.Errorf("failed to delete Service: %w", delErr))
		}
	} else if !apierrors.IsNotFound(err) {
		allErrs = append(allErrs, fmt.Errorf("failed to get Service: %w", err))
	}

	// NOT REQUIRED, IF SERVICE GETS DELETED THE SLICE WILL BE DELETED AUTOMATICALLY
	// eps := &discoveryv1.EndpointSlice{}
	// err = r.Get(ctx, client.ObjectKey{Name: subResourcePrefix + rpe.Name, Namespace: rpe.Namespace}, eps)
	// if err == nil {
	// 	if delErr = r.Delete(ctx, eps); delErr != nil && !apierrors.IsNotFound(err) {
	// 		allErrs = append(allErrs, fmt.Errorf("failed to delete Endpoints: %w", delErr))
	// 	}
	// } else if !apierrors.IsNotFound(err) {
	// 	allErrs = append(allErrs, fmt.Errorf("failed to get Endpoints: %w", err))
	// }

	ing := &networkingv1.Ingress{}
	err = r.Get(ctx, client.ObjectKey{Name: subResourcePrefix + rpe.Name, Namespace: rpe.Namespace}, ing)
	if err == nil {
		if delErr = r.Delete(ctx, ing); delErr != nil && !apierrors.IsNotFound(err) {
			allErrs = append(allErrs, fmt.Errorf("failed to delete Ingress: %w", delErr))
		}
	} else if !apierrors.IsNotFound(err) {
		allErrs = append(allErrs, fmt.Errorf("failed to get Ingress: %w", err))
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("cleanup failed: %v", allErrs)
	}
	return nil
}
