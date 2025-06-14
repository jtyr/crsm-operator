package utils

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LabelSelectorPredicate defines custom predicate to reconcile only resources with matching labels.
func LabelSelectorPredicate(selector labels.Selector) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return selector.Matches(labels.Set(e.Object.GetLabels()))
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return selector.Matches(labels.Set(e.ObjectNew.GetLabels()))
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return selector.Matches(labels.Set(e.Object.GetLabels()))
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return selector.Matches(labels.Set(e.Object.GetLabels()))
		},
	}
}

// LabelsChangedPredicate defines custom predicate to reconcile only if resource labels changed.
func LabelsChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldLabels := e.ObjectOld.GetLabels()
			newLabels := e.ObjectNew.GetLabels()

			return !reflect.DeepEqual(oldLabels, newLabels)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// FinalizersChangedPredicate defines custom predicate to reconcile only if resource finalizers changed.
func FinalizersChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldFinalizers := e.ObjectOld.GetFinalizers()
			newFinalizers := e.ObjectNew.GetFinalizers()

			return !reflect.DeepEqual(oldFinalizers, newFinalizers)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// NamespaceLabelSelectorPredicate defines custom predicate to reconcile only
// resources within Namespaces with matching labels.
func NamespaceLabelSelectorPredicate(client client.Client, selector labels.Selector) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return namespaceMatches(client, selector, e.Object.GetNamespace())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return namespaceMatches(client, selector, e.ObjectNew.GetNamespace())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return namespaceMatches(client, selector, e.Object.GetNamespace())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return namespaceMatches(client, selector, e.Object.GetNamespace())
		},
	}
}

// namespaceMatches checks if the Namespace selector matches the Namespace labels.
func namespaceMatches(client client.Client, selector labels.Selector, namespace string) bool {
	var ns corev1.Namespace

	err := client.Get(context.Background(), types.NamespacedName{Name: namespace, Namespace: ""}, &ns)
	if err != nil {
		// Ignore missing Namespace
		return false
	}

	return selector.Matches(labels.Set(ns.GetLabels()))
}
