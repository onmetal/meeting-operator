/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
