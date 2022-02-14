package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/angelbarrera92/nginx-ingress-controller-configuration-checker/pkg/kube"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	nginxIngressControllerCheckerExample = `
	# check the current configuration of the internal-nginx-ingress-controller deployed as a deployment
	%[1]s %[2]s deploy internal-nginx-ingress-controller

	# check the current configuration of the nginx-ingress-controller deployed as a deployment
	%[1]s %[2]s deployment nginx-ingress-controller

	# check the current configuration of the custom-nginx-ingress-controller deployed as a daemonset
	%[1]s %[2]s ds custom-nginx-ingress-controller

	# check the current configuration of the internal-nginx-ingress-controller deployed as a daemonset
	%[1]s %[2]s daemonset internal-nginx-ingress-controller
`
)

// NginxIngressControllerChecker provides information required to check
// the current nginx ingress controller configuration
type NginxIngressControllerCheckerOptions struct {
	ctx                                  *context.Context
	configFlags                          *genericclioptions.ConfigFlags
	resource, name, container, namespace string

	args []string

	genericclioptions.IOStreams

	// Kube client
	rawConfig  api.Config
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
	dynamic    *dynamic.Interface
}

// NewNginxIngressControllerCheckerOptions provides an instance of NginxIngressControllerCheckerOptions with default values
func NewNginxIngressControllerCheckerOptions(streams genericclioptions.IOStreams) *NginxIngressControllerCheckerOptions {
	return &NginxIngressControllerCheckerOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdNginxIngressControllerChecker provides a cobra command wrapping NginxIngressControllerCheckerOptions
func NewCmdNginxIngressControllerChecker(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewNginxIngressControllerCheckerOptions(streams)
	command := "nginx-ingress-controller-configuration-checker"

	cmd := &cobra.Command{
		Use:          fmt.Sprintf("%s [kind] [name]", command),
		Short:        "Inspect and look for configuration drift in the nginx-ingress-controller running containers",
		Example:      fmt.Sprintf(nginxIngressControllerCheckerExample, "kubectl", command),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&o.container, "container", "c", "", "Container name. If omitted, the first container in the pod will be chosen")
	// TODO. Add option to dump the generated nginx.conf configuration (maybe something like -o nginx.conf)
	// TODO. Implement verbose mode

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for updating the current context
func (o *NginxIngressControllerCheckerOptions) Complete(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	o.ctx = &ctx

	if len(args) != 2 {
		return errors.New("specify both a resource and its name")
	}
	o.args = args

	o.resource = o.args[0]
	o.name = o.args[1]

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	o.namespace, err = cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	// Kube client
	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}
	o.restConfig = restConfig

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.clientset = clientset

	dynamic, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.dynamic = &dynamic

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *NginxIngressControllerCheckerOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errors.New("no current context is set")
	}

	if len(o.namespace) == 0 {
		return errors.New("specify a namespace")
	}

	if len(o.resource) == 0 {
		return errors.New("specify a resource")
	}

	if len(o.name) == 0 {
		return errors.New("specify a name")
	}

	return nil
}

// Actually looks for configuration drift in the nginx-ingress-controller running containers
func (o *NginxIngressControllerCheckerOptions) Run() error {
	var p kube.PodController

	switch o.resource {
	case "deployment", "deploy":
		deploy, err := o.clientset.AppsV1().Deployments(o.namespace).Get(*o.ctx, o.name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		fmt.Fprint(o.Out, "deployment found: ", deploy.Name, "\n")

		p = kube.NewDeploymentPodController(o.ctx, o.clientset, deploy)
	case "daemonset", "ds":
		daemonSet, err := o.clientset.AppsV1().DaemonSets(o.namespace).Get(*o.ctx, o.name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		fmt.Fprint(o.Out, "daemonset found: ", daemonSet.Name, "\n")

		p = kube.NewDaemonsetPodController(o.ctx, o.clientset, daemonSet)
	}

	if p == nil {
		return errors.New("unsupported resource")
	}

	podList, err := p.List()
	if err != nil {
		return err
	}

	err = o.checkConfigurationDrift(podList)
	if err != nil {
		return err
	}

	return nil
}

func (o *NginxIngressControllerCheckerOptions) checkConfigurationDrift(podList *v1.PodList) error {
	// Map containing podName and nginx configuration
	nginxConfigurationPerPod := make(map[string]string)

	fmt.Fprintf(o.Out, "downloading configuration...\n")

	// channel to receive pod configuration
	downloadResults := make(chan downloadResult, len(podList.Items))

	// Invoke the goroutines to download the configuration
	for _, pod := range podList.Items {
		go download(o.restConfig, o.clientset, &pod, o.container, downloadResults)
	}

	// Wait for the goroutines to finish
	for _, pod := range podList.Items {
		dr := <-downloadResults
		if len(dr.stderr) != 0 {
			return fmt.Errorf((string)(dr.stderr))
		}
		if dr.err != nil {
			return dr.err
		}
		// TODO. Add to the verbose mode
		// fmt.Fprintf(o.Out, "configuration downloaded for pod %v. Length: %v\n", pod.Name, len(dr.content))
		nginxConfigurationPerPod[pod.Name] = dr.content
	}

	fmt.Fprintf(o.Out, "formatting nginx configuration files to avoid false positives...\n")

	// Create a channel to receive the results from the goroutines
	formatChannel := make(chan string, len(nginxConfigurationPerPod))

	// Invoke the goroutines to check the configuration drift
	for _, podConfig := range nginxConfigurationPerPod {
		go format(podConfig, formatChannel)
	}

	// Wait for the goroutines to finish
	for podName := range nginxConfigurationPerPod {
		nginxConfigurationPerPod[podName] = <-formatChannel
	}

	fmt.Fprintf(o.Out, "checking configuration drift...\n")

	// Create a channel to receive the results from the goroutines
	checkChannel := make(chan error, len(nginxConfigurationPerPod))

	// Invoke the goroutines to check the configuration drift
	for podName, podConfig := range nginxConfigurationPerPod {
		go check(podName, podConfig, nginxConfigurationPerPod, checkChannel)
	}

	// Wait for the goroutines to finish
	for range nginxConfigurationPerPod {
		err := <-checkChannel
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(o.Out, "no configuration drift detected.\n")

	return nil
}

type downloadResult struct {
	content string

	err    error
	stderr []byte
}

func download(restConfig *rest.Config, clientset *kubernetes.Clientset, pod *v1.Pod, containerName string, downloadChannel chan downloadResult) {
	// Get the container from the list of containers in the pod
	var container *v1.Container
	if containerName != "" {
		for _, c := range pod.Spec.Containers {
			if c.Name == containerName {
				container = &c
				break
			}
		}
	} else {
		container = &pod.Spec.Containers[0]
	}

	// Fail if no container is found
	if container == nil {
		downloadChannel <- downloadResult{
			content: "",
			err:     errors.New("container not found"),
			stderr:  nil,
		}
	} else {
		// Here we will save the content of the nginx.conf file
		nginxConfBuffer := bytes.NewBufferString("")

		//Create a new PodExec for this pod
		pe := kube.NewPodExec(pod, restConfig, clientset, nil, nginxConfBuffer)

		// Execute the cat command to actually get the nginx.conf file
		stderr, err := pe.Exec(container.Name, []string{"cat", "/etc/nginx/nginx.conf"})

		downloadChannel <- downloadResult{
			content: nginxConfBuffer.String(),
			err:     err,
			stderr:  stderr,
		}
	}
}

func format(nginxConf string, formatChannel chan string) {
	reg := regexp.MustCompile(`(?m)(^[\t]+#|^#).*`)
	fortmattedNginxConf := reg.ReplaceAllString(nginxConf, "")
	formatChannel <- fortmattedNginxConf
}

func check(podName string, nginxConfiguration string, nginxConfigurationPerPod map[string]string, checkChannel chan error) {
	var err error

	for otherPodName, otherPodConfig := range nginxConfigurationPerPod {
		if nginxConfiguration != otherPodConfig {
			err = fmt.Errorf("%s has a different configuration than %s", podName, otherPodName)
			break
		}
	}

	checkChannel <- err
}
