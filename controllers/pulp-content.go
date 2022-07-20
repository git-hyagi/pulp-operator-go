/*
Copyright 2022.

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

package controllers

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) pulpContentController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// Controller Deployment
	cntDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content", Namespace: pulp.Namespace}, cntDeployment)
	newCntDeployment := r.deploymentForPulpContent(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Content Deployment", "Deployment.Namespace", newCntDeployment.Namespace, "Deployment.Name", newCntDeployment.Name)
		err = r.Create(ctx, newCntDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Content Deployment", "Deployment.Namespace", newCntDeployment.Namespace, "Deployment.Name", newCntDeployment.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Content Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if !equality.Semantic.DeepDerivative(newCntDeployment.Spec, cntDeployment.Spec) {
		log.Info("The Content Deployment has been modified! Reconciling ...")
		err = r.Update(ctx, newCntDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the Content Deployment object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// SERVICE
	cntSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content-svc", Namespace: pulp.Namespace}, cntSvc)
	newCntSvc := r.serviceForContent(pulp)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		log.Info("Creating a new Content Service", "Service.Namespace", newCntSvc.Namespace, "Service.Name", newCntSvc.Name)
		err = r.Create(ctx, newCntSvc)
		if err != nil {
			log.Error(err, "Failed to create new Content Service", "Service.Namespace", newCntSvc.Namespace, "Service.Name", newCntSvc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Content Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if !equality.Semantic.DeepDerivative(newCntSvc.Spec, cntSvc.Spec) {
		log.Info("The Content Service has been modified! Reconciling ...")
		err = r.Update(ctx, newCntSvc)
		if err != nil {
			log.Error(err, "Error trying to update the Content Service object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpContent returns a pulp-content Deployment object
func (r *PulpReconciler) deploymentForPulpContent(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	runAsUser := int64(0)
	fsGroup := int64(0)

	ls := labelsForPulpContent(m)
	replicas := m.Spec.Content.Replicas
	resources := m.Spec.Content.ResourceRequirements

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-content",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       m.Spec.DeploymentType + "-content",
				"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-content-" + m.Name,
				"app.kubernetes.io/component":  "content",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:     "quay.io/pulp/pulp",
						Name:      "content",
						Args:      []string{"pulp-content"},
						Resources: resources,
						Env: []corev1.EnvVar{
							{
								Name:  "POSTGRES_SERVICE_HOST",
								Value: m.Name + "-database-svc." + m.Namespace + ".svc",
							},
							{
								Name:  "POSTGRES_SERVICE_PORT",
								Value: "5432",
							},
							{
								Name:  "PULP_GUNICORN_TIMEOUT",
								Value: "60",
							},
							{
								Name:  "PULP_API_WORKERS",
								Value: "1",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 24816,
							Protocol:      "TCP",
						}},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      m.Name + "-server",
								MountPath: "/etc/pulp/settings.py",
								SubPath:   "settings.py",
								ReadOnly:  true,
							},
							{
								Name:      m.Name + "-admin-password",
								MountPath: "/etc/pulp/pulp-admin-password",
								SubPath:   "admin-password",
								ReadOnly:  true,
							},
							{
								Name:      m.Name + "-db-fields-encryption",
								MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
								SubPath:   "database_fields.symmetric.key",
								ReadOnly:  true,
							},
							{
								Name:      "file-storage-tmp",
								MountPath: "/var/lib/pulp/tmp",
								ReadOnly:  false,
							},
						},
					}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
						FSGroup:   &fsGroup,
					},
					Volumes: []corev1.Volume{
						{
							Name: m.Name + "-admin-password",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: m.Name + "-admin-password",
									Items: []corev1.KeyToPath{{
										Path: "admin-password",
										Key:  "password",
									}},
								},
							},
						},
						{
							Name: m.Name + "-server",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: m.Name + "-server",
									Items: []corev1.KeyToPath{{
										Key:  "settings.py",
										Path: "settings.py",
									}},
								},
							},
						},
						{
							Name: m.Name + "-db-fields-encryption",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: m.Name + "-db-fields-encryption",
									Items: []corev1.KeyToPath{{
										Key:  "database_fields.symmetric.key",
										Path: "database_fields.symmetric.key",
									}},
								},
							},
						},
						{
							Name: "file-storage-tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForPulpContent returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpContent(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-content",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-content-" + m.Name,
		"app.kubernetes.io/component":  "content",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"app":                          "pulp-content",
		"pulp_cr":                      m.Name,
	}
}

// serviceForContent returns a service object for pulp-content
func (r *PulpReconciler) serviceForContent(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceContentObject(m.Name, m.Namespace)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceContentObject(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-content-svc",
			Namespace: namespace,
		},
		Spec: serviceContentSpec(name),
	}
}

// content service spec
func serviceContentSpec(name string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24816}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		ClusterIP:             "None",
		ClusterIPs:            []string{"None"},
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Port:       24816,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app":     "pulp-content",
			"pulp_cr": name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}
