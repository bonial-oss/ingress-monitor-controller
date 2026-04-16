package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// DefaultProvider is the provider that is used if none is specified
	// explicitly.
	DefaultProvider = ProviderSite24x7

	// DefaultNameTemplate is the default template used for naming monitors.
	DefaultNameTemplate = "{{.Namespace}}-{{.IngressName}}"
)

// Options holds the options that can be configured via cli flags.
type Options struct {
	ProviderConfigFile string
	Namespace          string
	ProviderName       string
	NameTemplate       string
	NoDelete           bool
	CreationDelay      time.Duration
	EnableHTTPRoute    bool
	ProviderConfig     ProviderConfig
}

// NewDefaultOptions creates a new *Options value with defaults set.
func NewDefaultOptions() *Options {
	return &Options{
		ProviderName:   DefaultProvider,
		NameTemplate:   DefaultNameTemplate,
		ProviderConfig: NewDefaultProviderConfig(),
	}
}

// AddFlags adds cli flags for configurable options to the command.
func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.NoDelete, "no-delete", o.NoDelete, "If set, monitors will not be deleted if the ingress is deleted.")
	cmd.Flags().DurationVar(&o.CreationDelay, "creation-delay", o.CreationDelay, "Duration to wait after an ingress is created before creating the monitor for it.")
	cmd.Flags().StringVar(&o.NameTemplate, "name-template", o.NameTemplate, "The template to use for the monitor name. Valid fields are: .IngressName, .Namespace.")
	cmd.Flags().StringVar(&o.Namespace, "namespace", o.Namespace, "Namespace to watch. If empty, all namespaces are watched.")
	cmd.Flags().StringVar(&o.ProviderConfigFile, "provider-config", o.ProviderConfigFile, "Location of the config file for the monitor providers.")
	cmd.Flags().BoolVar(&o.EnableHTTPRoute, "enable-httproute", o.EnableHTTPRoute, "Enable watching Gateway API HTTPRoute resources for monitor creation.")
	cmd.Flags().StringVar(&o.ProviderName, "provider", o.ProviderName, "The provider to use for creating monitors.")
}

// Validate validates options.
func (o *Options) Validate() error {
	if o.CreationDelay < 0 {
		return errors.Errorf("--creation-delay has to be greater than or equal to 0s")
	}

	if o.NameTemplate == "" {
		return errors.Errorf("--name-template must not be empty")
	}

	if o.ProviderName == "" {
		return errors.Errorf("--provider must not be empty")
	}

	return nil
}
