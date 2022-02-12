package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
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
	configFlags *genericclioptions.ConfigFlags

	rawConfig api.Config
	args      []string

	genericclioptions.IOStreams
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

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for updating the current context
func (o *NginxIngressControllerCheckerOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *NginxIngressControllerCheckerOptions) Validate() error {
	return nil
}

// Actually looks for configuration drift in the nginx-ingress-controller running containers
func (o *NginxIngressControllerCheckerOptions) Run() error {
	return nil
}
