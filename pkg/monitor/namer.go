package monitor

import (
	"bytes"
	"text/template"

	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
)

type templateArgs struct {
	// Kind is the kind of the Kubernetes resource (e.g. "Ingress" or
	// "HTTPRoute").
	Kind string

	// Name is the name of the Kubernetes resource.
	Name string

	// IngressName is an alias for Name, kept for backward compatibility with
	// existing name templates.
	IngressName string

	Namespace string
}

// Namer builds names for ingress monitors from a name template.
type Namer struct {
	template *template.Template
}

// NewNamer creates a new *Namer with given name template string. Returns an
// error if the name template is invalid.
func NewNamer(nameTemplate string) (*Namer, error) {
	tpl, err := template.New("monitor-name").Parse(nameTemplate)
	if err != nil {
		return nil, err
	}

	n := &Namer{
		template: tpl,
	}

	return n, nil
}

// Name builds a monitor name for the given source. Returns an error if
// rendering the name template fails.
func (n *Namer) Name(source models.MonitorSource) (string, error) {
	var buf bytes.Buffer

	err := n.template.Execute(&buf, templateArgs{
		Kind:        source.Kind,
		Name:        source.Name,
		IngressName: source.Name,
		Namespace:   source.Namespace,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
