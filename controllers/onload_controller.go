// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package controllers

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kmm "github.com/kubernetes-sigs/kernel-module-management/api/v1beta1"

	onloadv1alpha1 "github.com/Xilinx-CNS/kubernetes-onload/api/v1alpha1"
	"k8s.io/utils/ptr"
)

// OnloadReconciler reconciles a Onload object
type OnloadReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	DevicePluginImage string
}

//+kubebuilder:rbac:groups=onload.amd.com,resources=onloads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=onload.amd.com,resources=onloads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kmm.sigs.x-k8s.io,resources=modules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=onload.amd.com,resources=onloads/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="core",resources=nodes,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups="core",resources=pods,verbs=get;list;watch;patch;delete
//+kubebuilder:rbac:groups="core",resources=pods/eviction,verbs=create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Onload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *OnloadReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	onload := &onloadv1alpha1.Onload{}
	err := r.Get(ctx, req.NamespacedName, onload)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that
			// it was deleted or not created.
			// In this way, we will stop the reconciliation
			log.Info("Onload resource not found." +
				" Ignoring since object must be deleted")
			err := r.deleteLabels(ctx, req.NamespacedName)
			if err != nil {
				log.Error(err, "Failed to clean labels after deletion")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Onload")
		return ctrl.Result{}, err
	}

	// Check if the Onload instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isOnloadMarkedToBeDeleted := onload.GetDeletionTimestamp() != nil

	if isOnloadMarkedToBeDeleted {
		return ctrl.Result{}, nil
	}

	res, err := r.addKmmLabelsToNodes(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to add kmm label to Nodes")
		return ctrl.Result{}, err
	} else if res != nil {
		// Logging is handled in addKmmLabelsToNodes
		return *res, nil
	}

	res, err = r.createAndAddModules(ctx, onload)
	if err != nil {
		log.Info("Error creating module")
		return ctrl.Result{}, err
	} else if res != nil {
		log.Info("Added Module(s)")
		return *res, nil
	}

	res, err = r.addOnloadLabelsToNodes(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to add Onload label to nodes")
		return ctrl.Result{}, err
	} else if res != nil {
		// logging is handled in the function
		return *res, nil
	}

	res, err = r.createDevicePluginDaemonSet(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to create Device Plugin daemonset")
		return ctrl.Result{}, err
	} else if res != nil {
		log.Info("Added Onload Device Plugin DaemonSet")
		return *res, nil
	}

	res, err = r.handleUpdate(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to handle updates")
		return ctrl.Result{}, err
	}
	if res != nil {
		log.Info("Onload CR is updating")
		return *res, nil
	}

	return ctrl.Result{}, nil
}

const kmmModuleLabelPrefix = "kmm.node.kubernetes.io/version-module"
const onloadModuleNameSuffix = "-onload-module"
const sfcModuleNameSuffix = "-sfc-module"

func kmmOnloadLabelName(name, namespace string) string {
	return kmmModuleLabelPrefix + "." + namespace + "." + name + onloadModuleNameSuffix
}

func kmmSFCLabelName(name, namespace string) string {
	return kmmModuleLabelPrefix + "." + namespace + "." + name + sfcModuleNameSuffix
}

const onloadLabelPrefix = "onload.amd.com/"

func onloadLabelName(name, namespace string) string {
	return onloadLabelPrefix + namespace + "." + name
}

func (r *OnloadReconciler) listNodesWithLabels(ctx context.Context, labelString string) (corev1.NodeList, error) {
	selector, err := labels.Parse(labelString)
	if err != nil {
		return corev1.NodeList{}, err
	}

	nodes := corev1.NodeList{}
	opt := client.MatchingLabelsSelector{Selector: selector}
	err = r.List(ctx, &nodes, opt)
	if err != nil {
		return corev1.NodeList{}, err
	}
	return nodes, nil
}

func (r *OnloadReconciler) deleteLabelFromNode(ctx context.Context, node corev1.Node, label string) error {
	nodeCopy := node.DeepCopy()
	delete(node.Labels, label)
	return r.Patch(ctx, &node, client.MergeFrom(nodeCopy))
}

func (r *OnloadReconciler) deleteLabels(ctx context.Context, namespacedName types.NamespacedName) error {
	log := log.FromContext(ctx)

	deleteLabel := func(label string) error {
		nodes, err := r.listNodesWithLabels(ctx, label)
		if err != nil {
			return err
		}

		for _, node := range nodes.Items {
			err := r.deleteLabelFromNode(ctx, node, label)
			if err != nil {
				log.Error(err, "Failed to remove label from Node",
					"Node", node.Name, "label key", label)
			}
		}
		return nil
	}

	err := deleteLabel(kmmOnloadLabelName(namespacedName.Name, namespacedName.Namespace))
	if err != nil {
		return err
	}

	err = deleteLabel(kmmSFCLabelName(namespacedName.Name, namespacedName.Namespace))
	if err != nil {
		return err
	}

	err = deleteLabel(onloadLabelName(namespacedName.Name, namespacedName.Namespace))
	if err != nil {
		return err
	}

	return nil
}

func (r *OnloadReconciler) addKmmLabelsToNodes(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Figure out a list of nodes that match Onload's selector.
	onloadLabels := labels.FormatLabels(onload.Spec.Selector)
	nodes, err := r.listNodesWithLabels(ctx, onloadLabels)
	if err != nil {
		return nil, err
	}

	changesMade := false

	addKmmLabelToNode := func(node corev1.Node, labelKey string, podLabelSet labels.Set,
	) (*ctrl.Result, error) {
		if _, found := node.Labels[labelKey]; found {
			return nil, nil
		}

		// We can only add the SFC label if the Onload label matches
		// the Onload CR version. Otherwise, we are likely within the
		// upgrade workflow, where we soon delete old KMM labels and
		// add new ones, causing the SFC kernel module to reload
		// unnecessarily.
		if labelKey != kmmOnloadLabelName(onload.Name, onload.Namespace) &&
			node.Labels[kmmOnloadLabelName(onload.Name, onload.Namespace)] != onload.Spec.Onload.Version {
			return nil, nil
		}

		// Check that there isn't a lingering module pod.
		// This can be removed, but without it the upgrade process is more
		// concurrent (rather than rolling), which leads to more pod
		// restarts/failures. Both methods work, but this approach maintains
		// the rolling nature of the update (as much as is possible).
		pods, err := r.getPodsOnNode(ctx, podLabelSet, node.Name)
		if err != nil {
			return nil, err
		} else if len(pods) > 0 {
			log.Info("Lingering Module Pod(s)", "Node", node.Name)
			return &ctrl.Result{RequeueAfter: defaultRequeueTime}, nil
		}

		nodeCopy := node.DeepCopy()
		node.Labels[labelKey] = onload.Spec.Onload.Version
		err = r.Patch(ctx, &node, client.MergeFrom(nodeCopy))
		if err != nil {
			log.Error(err, "Failed to patch Node with new label")
			return nil, err
		}
		changesMade = true
		return nil, nil
	}

	for _, node := range nodes.Items {
		// Try add Onload labels
		res, err := addKmmLabelToNode(node,
			kmmOnloadLabelName(onload.Name, onload.Namespace),
			labels.Set{"kmm.node.kubernetes.io/module.name": onload.Name + onloadModuleNameSuffix})
		if res != nil || err != nil {
			return res, err
		}

		// Add SFC labels conditionally
		if onloadUsesSFC(onload) {
			res, err := addKmmLabelToNode(node,
				kmmSFCLabelName(onload.Name, onload.Namespace),
				labels.Set{"kmm.node.kubernetes.io/module.name": onload.Name + sfcModuleNameSuffix})
			if res != nil || err != nil {
				return res, err
			}
		}
	}

	if changesMade {
		log.Info("Labelled Nodes with kmm label")
		return &ctrl.Result{Requeue: true}, nil
	}

	// Remove stale kmm labels from nodes that don't match the module kind's selector.
	res, err := r.removeStaleLabels(ctx, onload, kmmOnloadLabelName(onload.Name, onload.Namespace))
	if res != nil || err != nil {
		return res, err
	}

	return r.removeStaleLabels(ctx, onload, kmmSFCLabelName(onload.Name, onload.Namespace))
}

func (r *OnloadReconciler) addOnloadLabelsToNodes(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	labels := labels.FormatLabels(onload.Spec.Selector)
	labelKey := onloadLabelName(onload.Name, onload.Namespace)
	kmmLabel := kmmOnloadLabelName(onload.Name, onload.Namespace)
	nodes, err := r.listNodesWithLabels(ctx, labels)
	if err != nil {
		return nil, err
	}

	changesMade := false

	for _, node := range nodes.Items {
		// Don't add the onload label onto a node either without kmm or with the
		// wrong onload module version.
		if node.Labels[kmmLabel] != onload.Spec.Onload.Version {
			continue
		}
		if _, found := node.Labels[labelKey]; !found {
			nodeCopy := node.DeepCopy()
			node.Labels[labelKey] = onload.Spec.Onload.Version
			err := r.Patch(ctx, &node, client.MergeFrom(nodeCopy))
			if err != nil {
				log.Error(err, "Failed to patch Node with new label",
					"Node", node.Name)
				return nil, err
			}
			changesMade = true
		}
	}

	if changesMade {
		log.Info("Labelled nodes with Onload label")
		return &ctrl.Result{Requeue: true}, nil
	}

	return r.removeStaleLabels(ctx, onload, labelKey)
}

func (r *OnloadReconciler) removeStaleLabels(ctx context.Context, onload *onloadv1alpha1.Onload, labelKey string) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	onloadLabels := labels.FormatLabels(onload.Spec.Selector)
	onloadSelector, err := labels.Parse(onloadLabels)
	if err != nil {
		log.Error(err, "Failed to parse Onload spec.selector into Selector object",
			"Onload", onload, "spec.selector", onload.Spec.Selector)
		return nil, err
	}

	nodes, err := r.listNodesWithLabels(ctx, labelKey)
	if err != nil {
		log.Error(err, "Failed to list nodes with label", "label", labelKey)
		return nil, err
	}

	changesMade := false
	for _, node := range nodes.Items {
		if !onloadSelector.Matches(labels.Set(node.Labels)) {
			err := r.deleteLabelFromNode(ctx, node, labelKey)
			if err != nil {
				log.Error(err, "Failed to remove stale label from Node",
					"Node", node, "Label", labelKey)
				return nil, err
			}
			changesMade = true
		}
	}

	if changesMade {
		log.Info("Deleted stale label from Nodes", "Label", labelKey)
		return &ctrl.Result{Requeue: true}, nil
	}

	return nil, nil
}

func (r *OnloadReconciler) getNodesToUpgrade(ctx context.Context, onload *onloadv1alpha1.Onload) ([]corev1.Node, error) {
	log := log.FromContext(ctx)
	nodesToUpgrade := []corev1.Node{}

	labelKey := kmmOnloadLabelName(onload.Name, onload.Namespace)
	nodes, err := r.listNodesWithLabels(ctx, labelKey)
	if err != nil {
		log.Error(err, "Failed to list Nodes with kmm label",
			"kmm label", labelKey)
		return nodesToUpgrade, err
	}

	for _, node := range nodes.Items {
		nodeOnloadVersion := node.Labels[labelKey]
		if nodeOnloadVersion != onload.Spec.Onload.Version {
			nodesToUpgrade = append(nodesToUpgrade, node)
		}
	}

	return nodesToUpgrade, nil
}

func (r *OnloadReconciler) handleDevicePluginUpdate(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)
	devicePlugin := &appsv1.DaemonSet{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      onload.Name + devicePluginNameSuffix,
			Namespace: onload.Namespace,
		},
		devicePlugin,
	)
	if err != nil {
		log.Error(err, "Failed to get Device Plugin", "Onload", onload)
		return nil, err
	}

	if devicePlugin.ObjectMeta.Labels[onloadVersionLabel] == onload.Spec.Onload.Version {
		// Nothing to be done, so return
		return nil, nil
	}

	oldDevicePlugin := devicePlugin.DeepCopy()
	devicePlugin.ObjectMeta.Labels[onloadVersionLabel] = onload.Spec.Onload.Version
	devicePlugin.Spec.Template.Spec.InitContainers[0].Image = onload.Spec.Onload.UserImage
	err = r.Patch(ctx, devicePlugin, client.MergeFrom(oldDevicePlugin))
	if err != nil {
		log.Error(err, "Failed to patch Device Plugin DaemonSet",
			"Device Plugin", devicePlugin)
		return nil, err
	}

	log.Info("Patched Device Plugin Daemonset", "Device Plugin", devicePlugin)
	return &ctrl.Result{Requeue: true}, nil
}

func (r *OnloadReconciler) patchModule(ctx context.Context, module *kmm.Module,
	onload *onloadv1alpha1.Onload, getKernelMap kernelMapperFn,
) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	if module.Spec.ModuleLoader.Container.Version == onload.Spec.Onload.Version {
		// Nothing to be done, so return
		return nil, nil
	}

	kernelMappings := []kmm.KernelMapping{}

	for _, kmapSpec := range onload.Spec.Onload.KernelMappings {
		kmap := getKernelMap(kmapSpec)
		if kmap == nil {
			continue
		}

		kernelMappings = append(kernelMappings, *kmap)
	}

	oldModule := module.DeepCopy()
	module.Spec.ModuleLoader.Container.Version = onload.Spec.Onload.Version
	module.Spec.ModuleLoader.Container.KernelMappings = kernelMappings

	err := r.Patch(ctx, module, client.MergeFrom(oldModule))
	if err != nil {
		log.Error(err, "Failed to patch Module", "Module", module)
		return nil, err
	}

	log.Info("Updated Module definition for upgrade", "Module", module)
	return &ctrl.Result{Requeue: true}, nil
}

func (r *OnloadReconciler) handleModuleUpdate(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	patchOrDeleteModule := func(isUsed bool, moduleName string, getKernelMap kernelMapperFn,
	) (*ctrl.Result, error) {
		module := &kmm.Module{}
		err := r.Get(ctx, types.NamespacedName{Name: moduleName, Namespace: onload.Namespace}, module)

		//
		// Below are possible module kind-related actions.
		//
		// +------------------------+--------+----------+
		// | Exists \ Should exist? |  Yes   |    No    |
		// +------------------------+--------+----------+
		// | Yes                    | Patch  | Delete   |
		// | No                     | Error* | Continue |
		// +------------------------+--------+----------+
		//
		// (*) Error because the reconciliation loop must check the existence
		// of the SFC module kind before it reaches this module update code.
		//
		if err == nil { // Module kind exists
			if isUsed {
				return r.patchModule(ctx, module, onload, getKernelMap)
			} else {
				return &ctrl.Result{Requeue: true}, r.Delete(ctx, module)
			}
		} else if apierrors.IsNotFound(err) {
			if isUsed {
				err := fmt.Errorf("module %s should exist", moduleName)
				return nil, err
			} else {
				return nil, nil
			}
		} else {
			log.Error(err, "Failed to get Module", "Module", moduleName)
			return nil, err
		}
	}

	// Onload module kind is always used for Onload CR, hence "true".
	res, err := patchOrDeleteModule(true, onload.Name+onloadModuleNameSuffix, onloadKernelMapper)
	if res != nil || err != nil {
		return res, err
	}

	return patchOrDeleteModule(onloadUsesSFC(onload), onload.Name+sfcModuleNameSuffix, sfcKernelMapper)
}

func (r *OnloadReconciler) handleNodeUpdate(ctx context.Context, onload *onloadv1alpha1.Onload, node corev1.Node) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Remove the onload label from the node
	onloadLabelName := onloadLabelName(onload.Name, onload.Namespace)
	onloadLabelVersion, found := node.Labels[onloadLabelName]
	if found && onloadLabelVersion != onload.Spec.Onload.Version {
		err := r.deleteLabelFromNode(ctx, node, onloadLabelName)
		if err != nil {
			log.Error(err, "Could not patch Node to remove Onload label",
				"Node", node.Name)
			return nil, err
		} else {
			log.Info("Removed Onload label from Node " + node.Name)
			return &ctrl.Result{Requeue: true}, nil
		}
	}

	// Check that the Device Plugin pod has terminated before continuing
	labelSet := labels.Set{
		"onload.amd.com/name": onload.Name + devicePluginNameSuffix,
	}
	pods, err := r.getPodsOnNode(ctx, labelSet, node.Name)
	if err != nil {
		log.Error(err, "Failed to get Device Plugin pod on Node "+node.Name)
		return nil, err
	}
	if len(pods) > 0 {
		log.Info("Device plugin pod still exists.")
		return &ctrl.Result{RequeueAfter: defaultRequeueTime}, nil
	}

	// Evict pods using the onload resource
	res, err := r.evictOnloadedPods(ctx, node)
	if err != nil || res != nil {
		return res, err
	}

	// Remove kmm labels
	err = r.deleteLabelFromNode(ctx, node, kmmSFCLabelName(onload.Name, onload.Namespace))
	if err != nil {
		return nil, err
	}

	// The Onload label must be the last to be removed. Once completed,
	// the node will be considered upgraded, and the reconciliation loop
	// will not enter this function again for the given upgrade.
	err = r.deleteLabelFromNode(ctx, node, kmmOnloadLabelName(onload.Name, onload.Namespace))
	if err != nil {
		return nil, err
	}

	// now everything should be deleted / cleaned up
	// Requeue and enter the reconciliation loop again to handle re-labelling
	// this node
	return &ctrl.Result{Requeue: true}, nil
}

const defaultRequeueTime = 5 * time.Second

func (r *OnloadReconciler) handleUpdate(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	res, err := r.handleDevicePluginUpdate(ctx, onload)
	if err != nil || res != nil {
		return res, err
	}

	res, err = r.handleModuleUpdate(ctx, onload)
	if err != nil || res != nil {
		return res, err
	}

	nodesToUpgrade, err := r.getNodesToUpgrade(ctx, onload)
	if err != nil {
		return nil, err
	}
	if len(nodesToUpgrade) == 0 {
		// Nothing to be done, so return
		return nil, nil
	}

	// Upgrade a single node at a time in alphabetical order.
	node := slices.MinFunc(nodesToUpgrade, func(a, b corev1.Node) int {
		return cmp.Compare(a.Name, b.Name)
	})

	log.Info("Updating Onload version", "Node", node.Name, "Onload", onload)
	return r.handleNodeUpdate(ctx, onload, node)
}

func (r *OnloadReconciler) getPodsOnNode(ctx context.Context, labelSelector map[string]string, nodeName string) ([]corev1.Pod, error) {
	log := log.FromContext(ctx)

	podList := corev1.PodList{}
	opt := client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(labelSelector)),
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}),
	}

	pods := []corev1.Pod{}

	err := r.List(ctx, &podList, &opt)
	if err != nil {
		log.Error(err, "Failed to list Pods with selector(s)")
		return pods, err
	}

	return podList.Items, nil
}

func (r *OnloadReconciler) getPodsUsingOnload(ctx context.Context, node corev1.Node) ([]corev1.Pod, error) {
	log := log.FromContext(ctx)

	podsUsingOnload := []corev1.Pod{}

	allPods, err := r.getPodsOnNode(ctx, map[string]string{}, node.Name)
	if err != nil {
		log.Error(err, "Failed to list Pods")
	}

	// Ideally this function should be using a field selector that matches both
	// node.Name and pods the resource request, unfortunately k8s doesn't
	// currently support multiple requirements in a field selector. This means
	// that we will have to filter the pods manually.
	// https://github.com/kubernetes-sigs/controller-runtime/blob/d5bc8734caccabddac6a1bea250b0b9d771d318d/pkg/cache/internal/cache_reader.go#L120
	// https://github.com/kubernetes-sigs/controller-runtime/blob/d5bc8734caccabddac6a1bea250b0b9d771d318d/pkg/internal/field/selector/utils.go#L27

	for _, pod := range allPods {
		for _, container := range pod.Spec.Containers {
			numOnloads := container.Resources.Requests.Name("amd.com/onload", resource.DecimalSI)
			if numOnloads != nil && numOnloads.CmpInt64(0) > 0 {
				podsUsingOnload = append(podsUsingOnload, pod)
				break
			}
		}
	}

	return podsUsingOnload, nil
}

func (r *OnloadReconciler) evictOnloadedPods(ctx context.Context, node corev1.Node) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	onloadedPods, err := r.getPodsUsingOnload(ctx, node)
	if err != nil {
		log.Error(err, "Failed to get Pods using Onload")
		return nil, err
	}

	if len(onloadedPods) == 0 {
		return nil, nil
	}

	changesMade := false

	for _, pod := range onloadedPods {
		if pod.GetDeletionTimestamp() != nil {
			// Pod is already being terminated, we can just continue
			continue
		}
		err := r.SubResource("eviction").Create(ctx, &pod, &policyv1.Eviction{})
		if err != nil {
			log.Error(err, "Could not create eviction", "Pod", pod.Name)
			return nil, err
		}
		changesMade = true
	}

	if changesMade {
		log.Info("Created evictions for Pods using Onload", "Node", node.Name)
	} else {
		log.Info("Waiting for Pods using Onload to terminate", "Node", node.Name)
	}
	return &ctrl.Result{RequeueAfter: defaultRequeueTime}, nil
}

type kernelMapperFn func(onloadv1alpha1.OnloadKernelMapping) *kmm.KernelMapping

func onloadKernelMapper(spec onloadv1alpha1.OnloadKernelMapping) *kmm.KernelMapping {

	var buildSpec *kmm.Build = nil
	if spec.Build != nil {
		buildSpec = &kmm.Build{
			BuildArgs:           make([]kmm.BuildArg, 0),
			DockerfileConfigMap: spec.Build.DockerfileConfigMap,
		}
		for _, buildArg := range spec.Build.BuildArgs {
			arg := kmm.BuildArg{
				Name:  buildArg.Name,
				Value: buildArg.Value,
			}
			buildSpec.BuildArgs = append(buildSpec.BuildArgs, arg)
		}
	}

	return &kmm.KernelMapping{
		Regexp:         spec.Regexp,
		ContainerImage: spec.KernelModuleImage,
		Build:          buildSpec,
	}
}

func sfcKernelMapper(spec onloadv1alpha1.OnloadKernelMapping) *kmm.KernelMapping {
	// If the SFC field is not provided, the controller doesn't manage
	// the SFC kernel module.
	if spec.SFC == nil {
		return nil
	}

	// Otherwise, it reuses the Onload image.
	kmap := onloadKernelMapper(spec)

	kmap.Build = nil
	return kmap
}

// Return true if any of the kernel mappings contain a non-nil SFC field.
func onloadUsesSFC(onload *onloadv1alpha1.Onload) bool {
	return slices.ContainsFunc(onload.Spec.Onload.KernelMappings,
		func(kmap onloadv1alpha1.OnloadKernelMapping) bool {
			return kmap.SFC != nil
		})
}

func (r *OnloadReconciler) createAndAddModules(
	ctx context.Context,
	onload *onloadv1alpha1.Onload,
) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	createAndAddModule := func(moduleName string,
		modprobeArg string, inTreeModuleToRemove string, getKernelMap kernelMapperFn,
	) (*ctrl.Result, error) {
		module := &kmm.Module{}
		err := r.Get(
			ctx,
			types.NamespacedName{
				Name:      moduleName,
				Namespace: onload.Namespace,
			},
			module,
		)
		if err == nil {
			return nil, nil
		}
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get Module", "Module", moduleName)
			return nil, err
		}

		module, err = createModule(onload, moduleName,
			modprobeArg, inTreeModuleToRemove, getKernelMap)
		if err != nil {
			log.Error(err, "createModule failure")
			return nil, err
		}

		err = ctrl.SetControllerReference(onload, module, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set owner of newly created Module")
			return nil, err
		}

		err = r.Create(ctx, module)
		if err != nil {
			log.Error(err, "Failed to create new Module")
			return nil, err
		}

		return &ctrl.Result{Requeue: true}, nil
	}

	res, err := createAndAddModule(onload.Name+onloadModuleNameSuffix, "onload", "", onloadKernelMapper)
	if res != nil || err != nil {
		return res, err
	}

	if onloadUsesSFC(onload) {
		return createAndAddModule(onload.Name+sfcModuleNameSuffix, "sfc", "sfc", sfcKernelMapper)
	}

	return nil, nil
}

func createModule(
	onload *onloadv1alpha1.Onload,
	moduleName string,
	modprobeArg string, inTreeModuleToRemove string, getKernelMap kernelMapperFn,
) (*kmm.Module, error) {

	kernelMappings := []kmm.KernelMapping{}

	for _, kmapSpec := range onload.Spec.Onload.KernelMappings {
		kmap := getKernelMap(kmapSpec)

		// We may not need to create a new Module kind for this mapping,
		// e.g. if this is SFC and the user has deployed their kernel
		// module managed outside the controller.
		if kmap == nil {
			continue
		}

		kernelMappings = append(kernelMappings, *kmap)
	}

	module := &kmm.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleName,
			Namespace: onload.Namespace,
		},
		Spec: kmm.ModuleSpec{
			ModuleLoader: kmm.ModuleLoaderSpec{
				ServiceAccountName: onload.Spec.ServiceAccountName,
				Container: kmm.ModuleLoaderContainerSpec{
					Modprobe: kmm.ModprobeSpec{
						ModuleName: modprobeArg,
						Parameters: []string{"--first-time"},
					},
					InTreeModuleToRemove: inTreeModuleToRemove,

					KernelMappings:  kernelMappings,
					ImagePullPolicy: onload.Spec.Onload.ImagePullPolicy,
					Version:         onload.Spec.Onload.Version,
				},
			},
			Selector: onload.Spec.Selector,
		},
	}

	return module, nil
}

const devicePluginNameSuffix = "-onload-device-plugin-ds"
const onloadVersionLabel = "onload-version"

func (r *OnloadReconciler) createDevicePluginDaemonSet(
	ctx context.Context, onload *onloadv1alpha1.Onload,
) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	devicePlugin := &appsv1.DaemonSet{}
	devicePluginName := onload.Name + devicePluginNameSuffix
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      devicePluginName,
			Namespace: onload.Namespace,
		},
		devicePlugin,
	)
	if err == nil {
		return nil, nil
	}
	if !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get Onload Device Plugin")
		return nil, err
	}

	hostOnloadPath := "/opt/onload"
	if onload.Spec.DevicePlugin.HostOnloadPath != nil {
		hostOnloadPath = *onload.Spec.DevicePlugin.HostOnloadPath
	}

	postStart := &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{
				"/bin/sh", "-c",
				fmt.Sprintf(`set -e;
				chcon --type container_file_t --recursive %s ||
				echo "chcon failed. System may not be SELinux enabled.";`, hostOnloadPath),
			},
		},
	}

	preStop := &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{
				"/bin/sh", "-c",
				fmt.Sprintf(`set -e;
				rm -r %s;`, hostOnloadPath),
			},
		},
	}

	devicePluginContainer := corev1.Container{
		Name:            "device-plugin",
		Image:           r.DevicePluginImage,
		ImagePullPolicy: onload.Spec.DevicePlugin.ImagePullPolicy,
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		// Lifecycle to manage the Onload files in the host.
		Lifecycle: &corev1.Lifecycle{
			PostStart: postStart,
			PreStop:   preStop,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/var/lib/kubelet/device-plugins",
				Name:      "kubelet-socket",
			},
			{
				MountPath: hostOnloadPath,
				Name:      "host-onload",
			},
		},
	}

	devicePluginArgs := []string{}
	if onload.Spec.DevicePlugin.MaxPodsPerNode != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-maxPods=%d", *onload.Spec.DevicePlugin.MaxPodsPerNode))
	}

	if onload.Spec.DevicePlugin.SetPreload != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-setPreload=%t", *onload.Spec.DevicePlugin.SetPreload))
	}

	if onload.Spec.DevicePlugin.MountOnload != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-mountOnload=%t", *onload.Spec.DevicePlugin.MountOnload))
	}

	if onload.Spec.DevicePlugin.HostOnloadPath != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-hostOnloadPath=%s",
				*onload.Spec.DevicePlugin.HostOnloadPath))
	}

	if onload.Spec.DevicePlugin.BaseMountPath != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-baseMountPath=%s",
				*onload.Spec.DevicePlugin.BaseMountPath))
	}

	if onload.Spec.DevicePlugin.BinMountPath != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-binMountPath=%s",
				*onload.Spec.DevicePlugin.BinMountPath))
	}

	if onload.Spec.DevicePlugin.LibMountPath != nil {
		devicePluginArgs = append(devicePluginArgs,
			fmt.Sprintf("-libMountPath=%s",
				*onload.Spec.DevicePlugin.LibMountPath))
	}

	if len(devicePluginArgs) > 0 {
		devicePluginContainer.Args = devicePluginArgs
	}

	workerContainerName := "onload-worker"

	workerContainerEnv := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "CONTAINER_NAME",
			Value: workerContainerName,
		},
		{
			Name:  "ONLOAD_CP_SERVER_PATH",
			Value: "/mnt/onload/sbin/onload_cp_server",
		},
	}

	workerContainer := corev1.Container{
		Name:            workerContainerName,
		Image:           r.DevicePluginImage,
		ImagePullPolicy: onload.Spec.DevicePlugin.ImagePullPolicy,

		Command: []string{
			"onload-worker",
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/mnt/onload",
				Name:      "worker-volume",
			},
		},
		Env: workerContainerEnv,
	}

	initContainer := corev1.Container{
		Name:            onload.Name + "-onload-device-plugin" + "-init",
		Image:           onload.Spec.Onload.UserImage,
		ImagePullPolicy: onload.Spec.Onload.ImagePullPolicy,
		Command: []string{
			"/bin/sh", "-c",
			// The Kubelet can be configured to store an emptyDir in the host's
			// filesystem. `mkdir -p` is used in the case of a node reboot the
			// emptyDir might be older than the actual pod, so the initContainer
			// shouldn't fail if the directory already exists.
			`set -e;
			cp -TRv /opt/onload /host/onload;

			mkdir -vp /mnt/onload/sbin/;
			cp -v /opt/onload/sbin/onload_cp_server /mnt/onload/sbin/;
			`,
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				// MounthPath here doesn't have to match hostOnloadPath. The
				// location in the initContainer's filesystem doesn't affect the
				// device plugin, only whether they are both looking at the same
				// volumeMount.
				MountPath: "/host/onload",
				Name:      "host-onload",
			},
			{
				MountPath: "/mnt/onload",
				Name:      "worker-volume",
			},
		},
	}

	dsLabels := map[string]string{"onload.amd.com/name": devicePluginName}

	kubeletSocketVolume := corev1.Volume{
		Name: "kubelet-socket",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/var/lib/kubelet/device-plugins",
				Type: ptr.To(corev1.HostPathDirectory),
			},
		},
	}

	hostOnloadVolume := corev1.Volume{
		Name: "host-onload",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: hostOnloadPath,
				Type: ptr.To(corev1.HostPathDirectoryOrCreate),
			},
		},
	}

	emptyDirVolume := corev1.Volume{
		Name: "worker-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: resource.NewQuantity(8*1024*1024, resource.BinarySI),
			},
		},
	}

	devicePlugin = &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      devicePluginName,
			Namespace: onload.Namespace,
			Labels:    map[string]string{onloadVersionLabel: onload.Spec.Onload.Version},
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType},
			Selector:       &metav1.LabelSelector{MatchLabels: dsLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: onload.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						devicePluginContainer,
						workerContainer,
					},
					Volumes: []corev1.Volume{
						kubeletSocketVolume,
						hostOnloadVolume,
						emptyDirVolume,
					},
					NodeSelector: onload.Spec.Selector,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      onloadLabelName(onload.Name, onload.Namespace),
												Operator: corev1.NodeSelectorOpExists,
												Values:   []string{},
											},
										},
									},
								},
							},
						},
					},
					InitContainers: []corev1.Container{initContainer},
					HostNetwork:    true,
					HostPID:        true,
					HostIPC:        true,
				},
			},
		},
	}

	err = controllerutil.SetControllerReference(onload, devicePlugin, r.Scheme)
	if err != nil {
		log.Error(err, "Failed to set controller reference for Device Plugin")
		return nil, err
	}

	err = r.Create(ctx, devicePlugin)
	if err != nil {
		log.Error(err, "Failed to create Onload Device Plugin")
		return nil, err
	} else {
		return &ctrl.Result{Requeue: true}, nil
	}
}

func (r *OnloadReconciler) nodeLabelWatchFunc(ctx context.Context, obj client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	onloadList := onloadv1alpha1.OnloadList{}
	err := r.List(ctx, &onloadList)
	if err != nil {
		return requests
	}

	// TODO: Check labels here to only requeue relevant onloads.

	for _, onload := range onloadList.Items {
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: onload.Name, Namespace: onload.Namespace}}
		requests = append(requests, request)
	}

	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&corev1.Pod{}, "spec.nodeName", func(o client.Object) []string {
			return []string{o.(*corev1.Pod).Spec.NodeName}
		})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&onloadv1alpha1.Onload{}).
		Owns(&kmm.Module{}).
		Owns(&appsv1.DaemonSet{}).
		Watches(&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.nodeLabelWatchFunc)).
		Complete(r)
}
