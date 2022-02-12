package main

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/angelbarrera92/nginx-ingress-controller-configuration-checker/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-nginx-ingress-controller-configuration-checker", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdNginxIngressControllerChecker(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
