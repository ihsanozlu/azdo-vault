package internal

import "strings"

// IsEndpointKey tells if a task/build input key usually contains a service connection id.
func IsEndpointKey(k string) bool {
	switch strings.ToLower(k) {
	case "containerregistry",
		"connectedservicename",
		"connectedservicenamearm",
		"azuresubscription",
		"azureresourcemanagerconnection",
		"dockerregistryserviceconnection",
		"kubernetesserviceendpoint",
		"externalendpoint",
		"serviceconnection",
		"endpoint",
		"subscription":
		return true
	default:
		return false
	}
}
