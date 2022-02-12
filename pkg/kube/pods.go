package kube

import v1 "k8s.io/api/core/v1"

type PodController interface {
	List() (*v1.PodList, error)
}
