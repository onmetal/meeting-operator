// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import v1 "k8s.io/api/core/v1"

type StorageSpec struct {
	EmptyDir *v1.EmptyDirVolumeSource `json:"empty_dir,omitempty"`
	PVC      v1.PersistentVolumeClaim `json:"pvc,omitempty"`
}
