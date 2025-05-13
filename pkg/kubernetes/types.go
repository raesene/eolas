package kubernetes

// ClusterConfig represents the top-level structure for Kubernetes cluster configuration
type ClusterConfig struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Items      []Item `json:"items,omitempty"`
}

// Item represents an individual Kubernetes resource
type Item struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

// Metadata contains resource metadata
type Metadata struct {
	Name              string            `json:"name,omitempty"`
	Namespace         string            `json:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty"`
}