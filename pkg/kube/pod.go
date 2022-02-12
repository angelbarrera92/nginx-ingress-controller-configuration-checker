package kube

import (
	"bytes"
	"fmt"
	"io"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type podExec struct {
	pod *v1.Pod

	config    *rest.Config
	clientset *kubernetes.Clientset

	stdin  io.Reader
	stdout io.Writer
}

// NewPodExec creates a new podExec
func NewPodExec(pod *v1.Pod, config *rest.Config, clientset *kubernetes.Clientset, stdin io.Reader, stdout io.Writer) *podExec {
	return &podExec{
		pod:       pod,
		config:    config,
		clientset: clientset,
		stdin:     stdin,
		stdout:    stdout,
	}
}

// Exec executes a command in a given container
func (pe *podExec) Exec(containerName string, command []string) ([]byte, error) {

	req := pe.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pe.pod.Name).
		Namespace(pe.pod.Namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding to scheme: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&v1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     pe.stdin != nil,
		Stdout:    pe.stdout != nil,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(pe.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("error while creating Executor: %v", err)
	}

	var stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  pe.stdin,
		Stdout: pe.stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return nil, fmt.Errorf("error in Stream: %v", err)
	}

	return stderr.Bytes(), nil
}
