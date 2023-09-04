// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package controllers

import (
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	Scheme *runtime.Scheme
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
		log.Info("Added Onload Module")
		return *res, nil
	}

	res, err = r.addOnloadLabelsToNodes(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to add Onload label to nodes")
		return ctrl.Result{}, err
	} else if res != nil {
		log.Info("Labelled nodes with Onload label")
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

func kmmLabelName(name, namespace string) string {
	return kmmModuleLabelPrefix + "." + namespace + "." + name + onloadModuleNameSuffix
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

func (r *OnloadReconciler) deleteLabels(ctx context.Context, namespacedName types.NamespacedName) error {
	log := log.FromContext(ctx)

	deleteLabel := func(label string) error {
		nodes, err := r.listNodesWithLabels(ctx, label)
		if err != nil {
			return err
		}

		for _, node := range nodes.Items {
			nodeCopy := node.DeepCopy()
			delete(node.Labels, label)
			err := r.Patch(ctx, &node, client.MergeFrom(nodeCopy))
			if err != nil {
				log.Error(err, "Failed to remove label from Node",
					"Node", node.Name, "label key", label)
			}
		}
		return nil
	}

	err := deleteLabel(kmmLabelName(namespacedName.Name, namespacedName.Namespace))
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
	kmmPodLabelSet := labels.Set{"kmm.node.kubernetes.io/module.name": onload.Name + onloadModuleNameSuffix}

	labels := labels.FormatLabels(onload.Spec.Selector)
	labelKey := kmmLabelName(onload.Name, onload.Namespace)

	nodes, err := r.listNodesWithLabels(ctx, labels)
	if err != nil {
		return nil, err
	}

	changesMade := false

	for _, node := range nodes.Items {
		if _, found := node.Labels[labelKey]; !found {
			// Check that there isn't a lingering module pod
			// This can be removed, but without it the upgrade process is more
			// concurrent (rather than rolling), which leads to more pod
			// restarts/failures. Both methods work, but this approach maintains
			// the rolling nature of the update (as much as is possible).
			pods, err := r.getPodsOnNode(ctx, kmmPodLabelSet, node.Name)
			if err != nil {
				return nil, err
			} else if len(pods) > 0 {
				log.Info("Lingering Module Pod on Node " + node.Name)
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
		}
	}

	var res *ctrl.Result = nil
	if changesMade {
		log.Info("Labelled Nodes with kmm label")
		res = &ctrl.Result{Requeue: true}
	}

	return res, nil
}

func (r *OnloadReconciler) addOnloadLabelsToNodes(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	labels := labels.FormatLabels(onload.Spec.Selector)
	labelKey := onloadLabelName(onload.Name, onload.Namespace)
	kmmLabel := kmmLabelName(onload.Name, onload.Namespace)
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

	var res *ctrl.Result = nil
	if changesMade {
		res = &ctrl.Result{Requeue: true}
	}

	return res, nil
}

func (r *OnloadReconciler) getNodesToUpgrade(ctx context.Context, onload *onloadv1alpha1.Onload) ([]corev1.Node, error) {
	log := log.FromContext(ctx)
	nodesToUpgrade := []corev1.Node{}

	labelKey := kmmLabelName(onload.Name, onload.Namespace)
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

func (r *OnloadReconciler) handleModuleUpdate(ctx context.Context, onload *onloadv1alpha1.Onload) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	module := &kmm.Module{}
	err := r.Get(ctx, types.NamespacedName{Name: onload.Name + onloadModuleNameSuffix, Namespace: onload.Namespace}, module)
	if err != nil {
		log.Error(err, "Failed to get Onload Module", "Onload", onload)
		return nil, err
	}

	if module.Spec.ModuleLoader.Container.Version == onload.Spec.Onload.Version {
		// Nothing to be done, so return
		return nil, nil
	}

	oldModule := module.DeepCopy()

	module.Spec.ModuleLoader.Container.Version = onload.Spec.Onload.Version

	for i := range module.Spec.ModuleLoader.Container.KernelMappings {
		moduleKmap := &module.Spec.ModuleLoader.Container.KernelMappings[i]
		for _, onloadKmap := range onload.Spec.Onload.KernelMappings {
			if onloadKmap.Regexp == moduleKmap.Regexp {
				moduleKmap.ContainerImage = onloadKmap.KernelModuleImage
				break
			}
		}
	}

	err = r.Patch(ctx, module, client.MergeFrom(oldModule))
	if err != nil {
		log.Error(err, "Failed to patch Module", "Module", module)
		return nil, err
	}

	log.Info("Updated Module definition for upgrade", "Module", module)
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

	// We just want to upgrade a single node at a time.
	// Since I don't know if there are any guarantees about how they are ordered
	// when returned from List(), we shall upgrade nodes by their name
	// alphabetically.
	// Golang 1.21 introduces the "slices" package which would allow us to find
	// the min element of a slice in a single line, however at the moment the
	// operator is written in Go 1.19 so this package isn't available.
	// In it's absence I am just using a local funcion.

	node := func(nodes []corev1.Node) corev1.Node {
		min := nodes[0]
		for _, node := range nodes {
			if node.Name < min.Name {
				min = node
			}
		}
		return min
	}(nodesToUpgrade)

	log.Info("Updating Onload version on Node "+node.Name, "Onload", onload)

	// Remove the onload label from the node
	onloadLabelName := onloadLabelName(onload.Name, onload.Namespace)
	onloadLabelVersion, found := node.Labels[onloadLabelName]
	if found && onloadLabelVersion != onload.Spec.Onload.Version {
		oldNode := node.DeepCopy()
		delete(node.Labels, onloadLabelName)
		err := r.Patch(ctx, &node, client.MergeFrom(oldNode))
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
	res, err = r.evictOnloadedPods(ctx, node)
	if err != nil || res != nil {
		return res, err
	}

	// Remove kmm label
	oldNode := node.DeepCopy()
	labelName := kmmLabelName(onload.Name, onload.Namespace)
	delete(node.Labels, labelName)
	err = r.Patch(ctx, &node, client.MergeFrom(oldNode))
	if err != nil {
		log.Error(err, "Could not patch node (removing kmm label) Node: "+node.Name)
		return nil, err
	}

	// now everything should be deleted / cleaned up
	// Requeue and enter the reconciliation loop again to handle re-labelling
	// this node
	return &ctrl.Result{Requeue: true}, nil
}

// This is just a temporary (TM) function since using a field selector isn't
// working (for our operator at the moment).
// By default the controller uses a caching client for reading operations from
// the cluster, but this isn't by default compatible with using a field selector
// My understanding is that adding an indexer should be sufficient to get this
// working, but it is a non-trivial fix and so it is left for later.
func (r *OnloadReconciler) getPodsOnNode(ctx context.Context, labelSelector map[string]string, nodeName string) ([]corev1.Pod, error) {
	log := log.FromContext(ctx)

	podList := corev1.PodList{}
	opt := client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(labelSelector)),
		// FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}),
	}

	pods := []corev1.Pod{}

	err := r.List(ctx, &podList, &opt)
	if err != nil {
		log.Error(err, "Failed to list Pods with field and label selectors")
		return pods, err
	}

	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (r *OnloadReconciler) getPodsUsingOnload(ctx context.Context, node corev1.Node) ([]corev1.Pod, error) {
	log := log.FromContext(ctx)

	allPods := corev1.PodList{}
	podsUsingOnload := []corev1.Pod{}

	// Ideally this func should be using a field selector, but that requires
	// using a non-caching reader.
	// For now we have to get all pods, then manually filter by nodeName and
	// resource requests.

	err := r.List(ctx, &allPods)
	if err != nil {
		log.Error(err, "Failed to list Pods")
	}

	for _, pod := range allPods.Items {
		if pod.Spec.NodeName != node.Name {
			continue
		}
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
		log.Info("Waiting for Pods using Onload to die", "Node", node.Name)
	}
	return &ctrl.Result{RequeueAfter: defaultRequeueTime}, nil
}

func (r *OnloadReconciler) createAndAddModules(
	ctx context.Context,
	onload *onloadv1alpha1.Onload,
) (*ctrl.Result, error) {
	log := log.FromContext(ctx)

	moduleNames := []string{
		onload.Name + onloadModuleNameSuffix,
		// onload.Name + "-sfc-module",
	}

	for _, moduleName := range moduleNames {
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
			continue
		}

		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get Onload Module")
			return nil, err
		}

		module, err = createModule(onload, "onload", moduleName)
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

	return nil, nil
}

func createModule(
	onload *onloadv1alpha1.Onload,
	modprobeArg string,
	moduleName string,
) (*kmm.Module, error) {

	kernelMappings := []kmm.KernelMapping{}

	for _, kmapSpec := range onload.Spec.Onload.KernelMappings {
		if kmapSpec.Build != nil {
			return nil, errors.New(
				"build keys in KernelMappings aren't currently supported",
			)
		}

		kmap := kmm.KernelMapping{
			Regexp:         kmapSpec.Regexp,
			ContainerImage: kmapSpec.KernelModuleImage,
		}
		kernelMappings = append(kernelMappings, kmap)
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

	devicePluginContainer := corev1.Container{
		Name:            "device-plugin",
		Image:           onload.Spec.DevicePlugin.DevicePluginImage,
		ImagePullPolicy: onload.Spec.DevicePlugin.ImagePullPolicy,
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/var/lib/kubelet/device-plugins",
				Name:      "kubelet-socket",
			},
			{
				MountPath: "/opt/onload",
				Name:      "host-onload",
			},
		},
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

	postStart := &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{
				"/bin/sh", "-c",
				`set -e;
				chcon --type container_file_t --recursive /opt/onload/;`,
			},
		},
	}

	preStop := &corev1.LifecycleHandler{
		Exec: &corev1.ExecAction{
			Command: []string{
				"/bin/sh", "-c",
				`set -e;
				rm -r /opt/onload;`,
			},
		},
	}

	workerContainer := corev1.Container{
		Name:            workerContainerName,
		Image:           onload.Spec.DevicePlugin.DevicePluginImage,
		ImagePullPolicy: onload.Spec.DevicePlugin.ImagePullPolicy,

		Command: []string{
			"onload-worker",
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/opt/onload",
				Name:      "host-onload",
			},
			{
				MountPath: "/mnt/onload",
				Name:      "worker-volume",
			},
		},

		// Lifecycle to manage the Onload files in the host.
		Lifecycle: &corev1.Lifecycle{
			PostStart: postStart,
			PreStop:   preStop,
		},
		Env: workerContainerEnv,
	}

	initContainer := corev1.Container{
		Name:            onload.Name + "-onload-device-plugin" + "-init",
		Image:           onload.Spec.Onload.UserImage,
		ImagePullPolicy: onload.Spec.Onload.ImagePullPolicy,
		Command: []string{
			"/bin/sh", "-c",
			`set -e;
			cp -TRv /opt/onload /host/onload;

			mkdir -v /mnt/onload/sbin/;
			cp -v /opt/onload/sbin/onload_cp_server /mnt/onload/sbin/;
			`,
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
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
				Path: "/opt/onload",
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&onloadv1alpha1.Onload{}).
		Owns(&kmm.Module{}).
		Owns(&appsv1.DaemonSet{}).
		Watches(&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.nodeLabelWatchFunc)).
		Complete(r)
}
