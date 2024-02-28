// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import v1 "k8s.io/api/core/v1"

type Port struct {
	Name     string      `json:"name"`
	Protocol v1.Protocol `json:"protocol,omitempty"`
	Port     int32       `json:"port,omitempty"`
}
