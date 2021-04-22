package utils

const (
	AppKubernetesPartOf = "jitsi-meet"
)

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func GetDefaultLabels(appName string) map[string]string {
	var defaultLabels = make(map[string]string)
	defaultLabels["app.kubernetes.io/appName"] = appName
	defaultLabels["app.kubernetes.io/part-of"] = AppKubernetesPartOf
	return defaultLabels
}
