package util

import (
	"fmt"
	"os"
)

const (
	// OperatorNameEnvVar is env variable for operator name
	OperatorNameEnvVar   = "OPERATOR_NAME"
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"
	// TODO: need to add this env var to charts
	OperatorNamespaceEnvVar = "MY_POD_NAMESPACE"
)

// GetOperatorName returns the operator name
func getEnvVar(envVar string) (string, error) {
	operatorName, found := os.LookupEnv(envVar)
	if !found {
		return "", fmt.Errorf("environment variable %s is not set", envVar)
	}
	if len(operatorName) == 0 {
		return "", fmt.Errorf("environment variable %s is empty", envVar)
	}
	return operatorName, nil
}

func GetOperatorName() (string, error) {
	return getEnvVar(OperatorNameEnvVar)
}

func GetWatchNamespace() (string, error) {
	return getEnvVar(WatchNamespaceEnvVar)
}

func GetOperatorNamespace() (string, error) {
	return getEnvVar(OperatorNamespaceEnvVar)
}
