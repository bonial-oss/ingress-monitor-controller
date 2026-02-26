package main

import (
	"flag"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/controller"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/monitor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	networkingv1 "k8s.io/api/networking/v1"
	runtime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	restconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	debug bool

	log = logf.Log.WithName("main")
)

// NewRootCommand creates a new *cobra.Command that is used as the root command
// for ingress-monitor-controller.
func NewRootCommand() *cobra.Command {
	options := config.NewDefaultOptions()

	cmd := &cobra.Command{
		Use:           "ingress-monitor-controller",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			runtime.SetLogger(zap.New(zap.UseDevMode(debug)))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Validate()
			if err != nil {
				return err
			}

			return Run(options)
		},
	}

	options.AddFlags(cmd)

	return cmd
}

func main() {
	cmd := NewRootCommand()

	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	cmd.PersistentFlags().BoolVar(&debug, "debug", debug, "Enable debug logging.")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Run sets up that controller and initiates the controller loop.
func Run(options *config.Options) error {
	if options.ProviderConfigFile != "" {
		log.V(1).Info("loading provider config", "config-file", options.ProviderConfigFile)

		providerConfig, err := config.ReadProviderConfig(options.ProviderConfigFile)
		if err != nil {
			return errors.Wrapf(err, "failed to load provider config from file")
		}

		err = mergo.Merge(&options.ProviderConfig, providerConfig, mergo.WithOverride)
		if err != nil {
			return errors.Wrapf(err, "failed to merge provider configs")
		}
	}

	mgr, err := manager.New(restconfig.GetConfigOrDie(), manager.Options{})
	if err != nil {
		return errors.Wrapf(err, "failed to create controller manager")
	}

	svc, err := monitor.NewService(options)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize monitor service")
	}

	reconciler := controller.NewIngressReconciler(mgr.GetClient(), svc, options)

	err = builder.
		ControllerManagedBy(mgr).
		Named("ingress-monitor-controller").
		For(&networkingv1.Ingress{}).
		Complete(reconciler)
	if err != nil {
		return errors.Wrapf(err, "failed to create controller")
	}

	err = mgr.Start(signals.SetupSignalHandler())
	if err != nil {
		return errors.Wrapf(err, "unable to run manager")
	}

	return nil
}
