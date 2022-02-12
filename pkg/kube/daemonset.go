package kube

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type daemonsetPodController struct {
	ctx       *context.Context
	kube      *kubernetes.Clientset
	daemonset *appsv1.DaemonSet
}

func NewDaemonsetPodController(ctx *context.Context, kube *kubernetes.Clientset, daemonset *appsv1.DaemonSet) *daemonsetPodController {
	return &daemonsetPodController{
		ctx:       ctx,
		kube:      kube,
		daemonset: daemonset,
	}
}

func (c *daemonsetPodController) List() (*v1.PodList, error) {
	return c.kube.CoreV1().Pods(c.daemonset.Namespace).List(*c.ctx, metav1.ListOptions{
		LabelSelector: labels.Set(c.daemonset.Spec.Template.Labels).AsSelector().String(),
	})
}
