// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package utils

const (
	AppKubernetesPartOf = "jitsi-meet"
)

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func GetDefaultLabelsForApp(appName string) map[string]string {
	defaultLabels := make(map[string]string)
	defaultLabels["app.kubernetes.io/appName"] = appName
	defaultLabels["app.kubernetes.io/part-of"] = AppKubernetesPartOf
	return defaultLabels
}
