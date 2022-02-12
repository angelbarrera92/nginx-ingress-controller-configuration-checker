package kube

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type deploymentPodController struct {
	ctx        *context.Context
	kube       *kubernetes.Clientset
	deployment *appsv1.Deployment
}

func NewDeploymentPodController(ctx *context.Context, kube *kubernetes.Clientset, deployment *appsv1.Deployment) *deploymentPodController {
	return &deploymentPodController{
		ctx:        ctx,
		kube:       kube,
		deployment: deployment,
	}
}

func (c *deploymentPodController) List() (*v1.PodList, error) {
	// List all pods in the deployment namespace with the labels from the deployment
	return c.kube.CoreV1().Pods(c.deployment.Namespace).List(*c.ctx, metav1.ListOptions{
		LabelSelector: labels.Set(c.deployment.Spec.Template.Labels).AsSelector().String(),
	})
}
