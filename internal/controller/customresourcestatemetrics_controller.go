/*
Copyright 2025.

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
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ksmv1 "github.com/jtyr/crsm-operator/api/v1"
	"github.com/jtyr/crsm-operator/internal/utils"
)

// Name of the finalizer that gets attached to the instance.
const FinalizerName = "ksm.jtyr.io/finalizer"

// Format for the begin marker.
const beginMarkerFormat = "# BEGIN CustomResourceStateMetrics %s"

// Format for the end marker.
const endMarkerFormat = "# END CustomResourceStateMetrics %s"

// Rype for the Ready status condition.
const conditionTypeReady = "Ready"

// Reasons for status conditions and events.
const reasonAdding = "Adding"
const reasonRemoving = "Removing"

// Logger definition with a prefix.
var log = ctrl.Log.WithName("[crsm]")

// CustomResourceStateMetricsReconciler reconciles a CustomResourceStateMetrics object
type CustomResourceStateMetricsReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Recorder          record.EventRecorder
	Selector          labels.Selector
	NamespaceSelector labels.Selector
}

// Data is a structure used to read the raw resources from the CustomResourceStateMetrics instance.
type Data struct {
	Resources []interface{} `yaml:"resources"`
}

//nolint:lll
// +kubebuilder:rbac:groups=ksm.jtyr.io,resources=customresourcestatemetrics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ksm.jtyr.io,resources=customresourcestatemetrics/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ksm.jtyr.io,resources=customresourcestatemetrics/finalizers,verbs=update

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *CustomResourceStateMetricsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// Content of the instance
	instance := &ksmv1.CustomResourceStateMetrics{}

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(
				err,
				"Unable to fetch",
				"instance", utils.NamespacedName(
					req.Name,
					req.Namespace))
		}

		// We'll ignore not-found errors, since they can't be fixed by
		// an immediate requeue (we'll need to wait for a new
		// notification), and we can get them on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Namespaced name of the instance
	instanceNamespacedName := utils.NamespacedName(instance.Name, instance.Namespace)

	if !instance.DeletionTimestamp.IsZero() { //nolint:gocritic
		log.Info("Deleting resources", "instance", instanceNamespacedName)

		// Record an event
		r.Recorder.Event(instance, corev1.EventTypeNormal, reasonRemoving, "Deleting resource.")

		// Remove instance from ConfigMap
		if err := r.deleteCustomResourceStateMetric(ctx, instance, instanceNamespacedName); err != nil {
			// Record the event
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonRemoving,
				"Failed to delete resources from the ConfigMap: %v", err)

			// Update the status condition
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    conditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  reasonRemoving,
				Message: "Failed to delete resources from the ConfigMap.",
			})
			if err := r.Status().Update(ctx, instance); err != nil {
				// Record the event
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonRemoving,
					"Failed to update status: %v", err)

				return ctrl.Result{}, fmt.Errorf("failed to update status for %s: %w",
					instanceNamespacedName, err)
			}

			return ctrl.Result{}, fmt.Errorf(
				"failed to delete resources from the ConfigMap for the CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		// Remove finalizer if it exists
		if controllerutil.ContainsFinalizer(instance, FinalizerName) {
			log.V(1).Info(
				"Deleting finalizer",
				"instance", instanceNamespacedName)

			controllerutil.RemoveFinalizer(instance, FinalizerName)

			if err := r.Update(ctx, instance); err != nil {
				// Record the event
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonRemoving,
					"Failed to delete finalizer: %v", err)

				// Update the status condition
				meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
					Type:    conditionTypeReady,
					Status:  metav1.ConditionFalse,
					Reason:  reasonRemoving,
					Message: "Failed to delete finalizer.",
				})
				if err := r.Status().Update(ctx, instance); err != nil {
					// Record the event
					r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonRemoving,
						"Failed to update status: %v", err)

					return ctrl.Result{}, fmt.Errorf("failed to update status for %s: %w",
						instanceNamespacedName, err)
				}

				return ctrl.Result{}, fmt.Errorf(
					"failed to delete finalizer from the CustomResourceStateMetrics instance %s: %w",
					instanceNamespacedName, err)
			}
		}
	} else if instance.Generation == 1 && !controllerutil.ContainsFinalizer(instance, FinalizerName) {
		log.Info("Creating resources", "instance", instanceNamespacedName)

		// Record the event
		r.Recorder.Event(instance, corev1.EventTypeNormal, reasonAdding, "Adding resources into the ConfigMap.")

		// Update the status condition
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    conditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  reasonAdding,
			Message: "Adding resources into the ConfigMap.",
		})
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf(
				"failed to update status for the CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		// Add resources
		if err := r.addCustomResourceStateMetric(ctx, instance, instanceNamespacedName); err != nil {
			// Record the event
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonAdding,
				"Failed to add resources into the ConfigMap: %v", err)

			// Update the status condition
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    conditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  reasonAdding,
				Message: "Failed to add resources into the ConfigMap.",
			})
			if err := r.Status().Update(ctx, instance); err != nil {
				// Record the event
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonAdding,
					"Failed to update status: %v", err)

				return ctrl.Result{}, fmt.Errorf("failed to update status for %s: %w",
					instanceNamespacedName, err)
			}

			return ctrl.Result{}, fmt.Errorf(
				"failed to add resources into the ConfigMap for CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		// Add finalizer if it doesn't exist yet
		if !controllerutil.ContainsFinalizer(instance, FinalizerName) {
			log.V(1).Info("Adding finalizer", "instance", instanceNamespacedName)

			controllerutil.AddFinalizer(instance, FinalizerName)

			// This triggers a new reconciliation
			if err := r.Update(ctx, instance); err != nil {
				// Record the event
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonAdding,
					"Failed to add finalizer: %v", err)

				return ctrl.Result{}, fmt.Errorf(
					"failed to add finalizer for the CustomResourceStateMetrics instance %s: %w",
					instanceNamespacedName, err)
			}
		}
	} else {
		log.Info("Updating resources", "instance", instanceNamespacedName)

		// Record the event
		r.Recorder.Event(instance, "Normal", reasonAdding, "Updating resources in the ConfigMap.")

		// Update resources
		if err := r.addCustomResourceStateMetric(ctx, instance, instanceNamespacedName); err != nil {
			// Record the event
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonAdding,
				"Failed to update the ConfigMap: %v", err)

			// Update the status condition
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    conditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  reasonAdding,
				Message: "Failed to update the ConfigMap.",
			})
			if err := r.Status().Update(ctx, instance); err != nil {
				// Record the event
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, reasonAdding,
					"Failed to update status: %v", err)

				return ctrl.Result{}, fmt.Errorf(
					"failed to update status for the CustomResourceStateMetrics instance %s: %w",
					instanceNamespacedName, err)
			}

			return ctrl.Result{}, fmt.Errorf(
				"failed to update resources for CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}
	}

	return ctrl.Result{}, nil
}

// deleteCustomResourceStateMetric removes resources from a ConfigMap.
func (r *CustomResourceStateMetricsReconciler) deleteCustomResourceStateMetric(
	ctx context.Context, instance *ksmv1.CustomResourceStateMetrics, instanceNamespacedName string) error {
	log.V(1).Info("Processing deletion of resources", "instance", instanceNamespacedName)

	// Define ConfigMap properties
	cmName := instance.Spec.ConfigMap.Name
	cmNamespace := instance.Spec.ConfigMap.Namespace
	cmKey := instance.Spec.ConfigMap.Key

	// If no Namespace was specified, use the namespace from the instance
	if cmNamespace == "" {
		cmNamespace = instance.Namespace
	}

	// Namespaced name of the ConfigMap
	cmNamespacedName := utils.NamespacedName(cmName, cmNamespace)

	// Check if the ConfigMap exists
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      cmName,
		Namespace: cmNamespace,
	}, cm)
	if err != nil {
		log.V(1).Info(
			"ConfigMap doesn't exist",
			"instance", instanceNamespacedName,
			"configMap", cmNamespacedName)

		// Record the event
		r.Recorder.Event(instance, corev1.EventTypeNormal, reasonRemoving,
			"The ConfigMap with the resources doesn't exist.")

		// Update the status condition
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    conditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  reasonRemoving,
			Message: "The ConfigMap with the resources doesn't exist.",
		})
		if err := r.Status().Update(ctx, instance); err != nil {
			return fmt.Errorf(
				"failed to update status for the CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		return nil
	}

	// Try to find the block in the ConfigMap
	lines := strings.Split(cm.Data[cmKey], "\n")
	found, beginIndex, endIndex := r.findBlock(instanceNamespacedName, lines)

	if !found {
		log.V(1).Info(
			"No block found",
			"instance", instanceNamespacedName,
			"configMap", cmNamespacedName)

		// Record the event
		r.Recorder.Event(instance, corev1.EventTypeNormal, reasonRemoving,
			"Resources don't exist in the ConfigMap.")

		// Update the status condition
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    conditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  reasonRemoving,
			Message: "Resources don't exist in the ConfigMap.",
		})
		if err := r.Status().Update(ctx, instance); err != nil {
			return fmt.Errorf(
				"failed to update status for the CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		return nil
	}

	log.V(1).Info(
		"Removing block",
		"instance", instanceNamespacedName,
		"configMap", cmNamespacedName,
		"position", fmt.Sprintf("%d;%d", beginIndex, endIndex))

	// Reset the current data and fill it with individual fragments
	// without the found block
	cm.Data[cmKey] = ""

	if beginIndex > 0 {
		cm.Data[cmKey] += r.joinLines(lines, 0, beginIndex-1)
	}

	if endIndex < len(lines)-1 {
		cm.Data[cmKey] += r.joinLines(lines, endIndex+1, -1)
	}

	// Update the ConfigMap
	if err := r.Update(ctx, cm); err != nil {
		return fmt.Errorf("failed to update the ConfigMap: %w", err)
	}

	// Record the event
	r.Recorder.Event(instance, corev1.EventTypeNormal, reasonRemoving,
		"Finished removal of resources from the ConfigMap.")

	// Update the status condition
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    conditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  reasonRemoving,
		Message: "Finished the removal of resources from the ConfigMap.",
	})
	if err := r.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf(
			"failed to update status for the CustomResourceStateMetrics instance %s: %w",
			instanceNamespacedName, err)
	}

	return nil
}

// addCustomResourceStateMetric adds resources into a ConfigMap.
func (r *CustomResourceStateMetricsReconciler) addCustomResourceStateMetric(
	ctx context.Context, instance *ksmv1.CustomResourceStateMetrics, instanceNamespacedName string) error {
	log.V(1).Info("Processing addition of reources", "instance", instanceNamespacedName)

	// Markers for the data separation in the final ConfigMap
	dataMarkerBegin := fmt.Sprintf("# BEGIN CustomResourceStateMetrics %s", instanceNamespacedName)
	dataMarkerEnd := fmt.Sprintf("# END CustomResourceStateMetrics %s", instanceNamespacedName)
	dataYaml, err := r.decodeData(instance.Spec.Resources)
	if err != nil {
		return fmt.Errorf("failed to decode resource data: %w", err)
	}

	// Define ConfigMap properties
	cmName := instance.Spec.ConfigMap.Name
	cmNamespace := instance.Spec.ConfigMap.Namespace
	cmKey := instance.Spec.ConfigMap.Key
	cmDataHeader := "kind: CustomResourceStateMetrics\nspec:\n  resources:\n"
	cmData := fmt.Sprintf(
		"%s\n%s%s\n",
		dataMarkerBegin,
		dataYaml,
		dataMarkerEnd,
	)

	// If no Namespace was specified, use the namespace from the instance
	if cmNamespace == "" {
		cmNamespace = instance.Namespace
	}

	// Namespaced name of the ConfigMap
	cmNamespacedName := utils.NamespacedName(cmName, cmNamespace)

	// Check if the ConfigMap exists
	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      cmName,
		Namespace: cmNamespace,
	}, cm)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get ConfigMap: %w", err)
		}

		// Create a new ConfigMap because it doesn't exist yet
		log.V(1).Info(
			"Creating a new ConfigMap",
			"instance", instanceNamespacedName,
			"configMap", cmNamespacedName)

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: cmNamespace,
			},
			Data: make(map[string]string),
		}

		cm.Data[cmKey] = cmDataHeader
		cm.Data[cmKey] += cmData

		if err := r.Create(ctx, cm); err != nil {
			return fmt.Errorf("failed to create a new ConfigMap: %w", err)
		}

		// Record the event
		r.Recorder.Event(instance, corev1.EventTypeNormal, reasonAdding,
			"Finished the addition of resources into a newly created ConfigMap.")

		// Update the status condition
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    conditionTypeReady,
			Status:  metav1.ConditionTrue,
			Reason:  reasonAdding,
			Message: "Finished the addition of resources into a newly created ConfigMap.",
		})
		if err := r.Status().Update(ctx, instance); err != nil {
			return fmt.Errorf(
				"failed to update status for the CustomResourceStateMetrics instance %s: %w",
				instanceNamespacedName, err)
		}

		return nil
	}

	log.V(1).Info(
		"Updating the existing ConfigMap",
		"instance", instanceNamespacedName,
		"configMap", cmNamespacedName)

	// Try to find the block in the ConfigMap
	lines := strings.Split(cm.Data[cmKey], "\n")
	found, beginIndex, endIndex := r.findBlock(instanceNamespacedName, lines)

	// Set the header if the ConfigMap is in its default state containing only the empty map
	if strings.TrimSpace(cm.Data[cmKey]) == "{}" {
		cm.Data[cmKey] = cmDataHeader
	}

	if found {
		if strings.TrimSuffix(cmData, "\n") == strings.Join(lines[beginIndex:endIndex+1], "\n") {
			log.V(1).Info(
				"The same block already exists",
				"instance", instanceNamespacedName,
				"configMap", cmNamespacedName,
				"position", fmt.Sprintf("%d;%d", beginIndex, endIndex))

			// Record the event
			r.Recorder.Event(instance, corev1.EventTypeNormal, reasonAdding,
				"The same resources already exist in the ConfigMap.")

			// Update the status condition
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    conditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  reasonAdding,
				Message: "The same resources already exist in the ConfigMap.",
			})
			if err := r.Status().Update(ctx, instance); err != nil {
				return fmt.Errorf(
					"failed to update status for the CustomResourceStateMetrics instance %s: %w",
					instanceNamespacedName, err)
			}

			return nil
		}

		log.V(1).Info(
			"Replacing existing block in the existing ConfigMap",
			"instance", instanceNamespacedName,
			"configMap", cmNamespacedName,
			"position", fmt.Sprintf("%d;%d", beginIndex, endIndex))

		// Reset the current data and fill it with individual fragments
		cm.Data[cmKey] = ""

		if beginIndex > 0 {
			cm.Data[cmKey] += r.joinLines(lines, 0, beginIndex-1)
		}

		cm.Data[cmKey] += cmData

		if endIndex < len(lines)-1 {
			cm.Data[cmKey] += r.joinLines(lines, endIndex+1, -1)
		}
	} else {
		log.V(1).Info(
			"Appending block at the end of the existing ConfigMap",
			"instance", instanceNamespacedName,
			"configMap", cmNamespacedName)

		cm.Data[cmKey] += cmData
	}

	// Update the ConfigMap
	if err := r.Update(ctx, cm); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	// Record the event
	r.Recorder.Event(instance, corev1.EventTypeNormal, reasonAdding,
		"Finished the addition of resources into an existing ConfigMap.")

	// Update the status condition
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    conditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  reasonAdding,
		Message: "Finished the addition of resources into an existing ConfigMap.",
	})
	if err := r.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf(
			"failed to update status for the CustomResourceStateMetrics instance %s: %w",
			instanceNamespacedName, err)
	}

	return nil
}

// decodeData decodes raw resources into YAML string.
func (r *CustomResourceStateMetricsReconciler) decodeData(resources []runtime.RawExtension) (string, error) {
	data := Data{}

	// Marshal raw portions of the resources into a structure
	for i := range resources {
		// Convert the raw structure to a JSON bytes array
		jsonBytes, err := resources[i].MarshalJSON()
		if err != nil {
			return "", fmt.Errorf("failed to encode resources #%d to JSON: %w", i, err)
		}

		// Convert the JSON bytes array to a structure
		var jsonObj interface{}
		err = json.Unmarshal(jsonBytes, &jsonObj)
		if err != nil {
			return "", fmt.Errorf("failed to decode resources #%d from JSON: %w", i, err)
		}

		data.Resources = append(data.Resources, jsonObj)
	}

	// Convert the data structure into YAML bytes array
	yamlData, err := yaml.Marshal(&data)
	if err != nil {
		return "", fmt.Errorf("failed to encode data to YAML: %w", err)
	}

	yamlDataString := string(yamlData)

	// Allow to remove the first line
	yamlDataSplit := strings.SplitN(yamlDataString, "\n", 2) //nolint:mnd

	// Return the original marshaled string if there is only one line
	if len(yamlDataSplit) < 2 { //nolint:mnd
		return yamlDataString, nil
	}

	// Retrurn the string without the first line
	return yamlDataSplit[1], nil
}

// findBlock finds a specific marker in the array of lines.
func (r *CustomResourceStateMetricsReconciler) findBlock(name string, lines []string) (bool, int, int) {
	found := false
	beginIndex := -1
	endIndex := -1

	beginMarker := fmt.Sprintf(beginMarkerFormat, name)
	endMarker := fmt.Sprintf(endMarkerFormat, name)

	for i, line := range lines {
		if line == beginMarker {
			beginIndex = i
		}

		if line == endMarker && beginIndex > -1 {
			endIndex = i
			found = true
		}
	}

	return found, beginIndex, endIndex
}

// joinLines joins slice of lines and makes sure the last line ends with a new
// line unless at the end of the lines.
func (r *CustomResourceStateMetricsReconciler) joinLines(lines []string, start, end int) string {
	strip := false
	lastIndex := len(lines) - 1

	if start < 0 {
		start = 0
	}

	if end == lastIndex {
		strip = true
	} else if end == -1 || end > lastIndex {
		end = lastIndex
		strip = true
	}

	result := strings.Join(lines[start:end+1], "\n")

	if strip {
		result = strings.TrimRight(result, "\n")
	} else if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomResourceStateMetricsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	combinedPredicate := predicate.And(
		// Reconcile only if generation value changed, labels or the finalizers changed
		predicate.Or(
			predicate.GenerationChangedPredicate{},
			utils.LabelsChangedPredicate(),
			utils.FinalizersChangedPredicate(),
		),
		// Label selectors must always match in order to reconcile
		utils.LabelSelectorPredicate(r.Selector),
		utils.NamespaceLabelSelectorPredicate(r.Client, r.NamespaceSelector),
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&ksmv1.CustomResourceStateMetrics{}).
		WithEventFilter(combinedPredicate).
		Named("customresourcestatemetrics").
		Complete(r)
}
