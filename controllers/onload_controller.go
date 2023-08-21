// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

package controllers

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kmm "github.com/kubernetes-sigs/kernel-module-management/api/v1beta1"

	onloadv1alpha1 "github.com/Xilinx-CNS/kubernetes-onload/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const controlPlaneScript = `
set -e;
echo /usr/bin/crictl | tee /sys/module/onload/parameters/cplane_server_path;
declare -r container_id=$(awk -F'[-./]' '/crio-/{print $(NF - 1); exit}' /proc/self/cgroup);
echo exec ${container_id} /opt/onload/sbin/onload_cp_server -K | tee /sys/module/onload/parameters/cplane_server_params;
sleep infinity;
`

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

	log.Info("Adding new Onload kind", "onload.Name",
		onload.Name, "onload.Namespace", onload.Namespace)

	ok, res, err := r.createAndAddModules(ctx, onload)
	if !ok {
		return res, err
	}

	err = r.createControlPlaneDaemonSet(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to create control plane daemon set")
		return ctrl.Result{}, err
	}

	err = r.createDevicePluginDaemonSet(ctx, onload)
	if err != nil {
		log.Error(err, "Failed to create device plugin daemonset")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OnloadReconciler) createAndAddModules(
	ctx context.Context,
	onload *onloadv1alpha1.Onload,
) (bool, ctrl.Result, error) {
	log := log.FromContext(ctx)

	moduleNames := []string{
		onload.Name + "-onload-module",
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
		if err != nil && apierrors.IsNotFound(err) {
			module, err := createModule(onload, "onload", moduleName)
			if err != nil {
				log.Error(err, "createModule failure")
				return false, ctrl.Result{}, err
			}

			err = ctrl.SetControllerReference(onload, module, r.Scheme)
			if err != nil {
				log.Error(err, "Failed to set owner of newly created module")
				return false, ctrl.Result{}, err
			}

			log.Info("Creating onload module")
			err = r.Create(ctx, module)
			if err != nil {
				log.Error(err, "Failed to create new module")
				return false, ctrl.Result{}, err
			}

			return false, ctrl.Result{Requeue: true}, nil

		} else if err != nil {
			log.Error(err, "Failed to get onload-module")
			return false, ctrl.Result{}, err
		}
	}

	return true, ctrl.Result{}, nil
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
				},
			},
			Selector: onload.Spec.Selector,
		},
	}

	return module, nil
}

func (r *OnloadReconciler) createControlPlaneDaemonSet(
	ctx context.Context, onload *onloadv1alpha1.Onload,
) error {
	log := log.FromContext(ctx)

	dsName := onload.Name + "-onload-cplane-ds"

	// Create the control plane daemon set only if it has not been created.
	ds := &appsv1.DaemonSet{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      dsName,
			Namespace: onload.Namespace,
		},
		ds,
	)

	if err == nil {
		log.Info("The control plane daemon set exists")
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	container := corev1.Container{
		Name:            onload.Name + "-onload-cplane",
		Image:           onload.Spec.Onload.UserImage,
		ImagePullPolicy: onload.Spec.Onload.ImagePullPolicy,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{controlPlaneScript},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
	}

	dsLabels := map[string]string{
		"onload.amd.com/name": dsName,
	}

	log.Info("Creating control plane daemon set")
	ds = &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: onload.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: dsLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: onload.Spec.ServiceAccountName,
					NodeSelector:       onload.Spec.Selector,
					HostNetwork:        true,
					HostPID:            true,
					HostIPC:            true,
					Containers:         []corev1.Container{container},
				},
			},
		},
	}

	err = controllerutil.SetControllerReference(onload, ds, r.Scheme)
	if err != nil {
		return err
	}

	return r.Create(ctx, ds)
}

func (r *OnloadReconciler) createDevicePluginDaemonSet(
	ctx context.Context, onload *onloadv1alpha1.Onload,
) error {
	log := log.FromContext(ctx)

	devicePlugin := &appsv1.DaemonSet{}
	devicePluginName := onload.Name + "-onload-device-plugin-ds"
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      devicePluginName,
			Namespace: onload.Namespace,
		},
		devicePlugin,
	)
	if err == nil {
		log.Info("Device plugin already exists.")
		return nil
	}
	if !apierrors.IsNotFound(err) {
		log.Error(err, "Could not find onload device plugin")
		return err
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
				rm -r /opt/onload /host/onload;`,
			},
		},
	}

	container := corev1.Container{
		Name:            onload.Name + "-onload-device-plugin",
		Image:           onload.Spec.DevicePlugin.DevicePluginImage,
		ImagePullPolicy: onload.Spec.DevicePlugin.ImagePullPolicy,
		Command:         []string{"/usr/bin/onload-plugin"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/var/lib/kubelet/device-plugins",
				Name:      "kubelet-socket",
			},
			{MountPath: "/opt/onload", Name: "host-onload"},
		},
		Lifecycle: &corev1.Lifecycle{
			PostStart: postStart,
			PreStop:   preStop,
		},
	}

	initContainer := corev1.Container{
		Name:            onload.Name + "-onload-device-plugin" + "-init",
		Image:           onload.Spec.Onload.UserImage,
		ImagePullPolicy: onload.Spec.Onload.ImagePullPolicy,
		Command: []string{
			"/bin/sh", "-c",
			`set -e;
			cp -TRv /opt/onload /host/onload;`,
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{MountPath: "/host/onload", Name: "host-onload"},
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

	devicePlugin = &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      devicePluginName,
			Namespace: onload.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: dsLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: onload.Spec.ServiceAccountName,
					Containers:         []corev1.Container{container},
					Volumes: []corev1.Volume{
						kubeletSocketVolume,
						hostOnloadVolume,
					},
					NodeSelector:   onload.Spec.Selector,
					InitContainers: []corev1.Container{initContainer},
				},
			},
		},
	}

	err = controllerutil.SetControllerReference(onload, devicePlugin, r.Scheme)
	if err != nil {
		return err
	}

	return r.Create(ctx, devicePlugin)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&onloadv1alpha1.Onload{}).
		Complete(r)
}
