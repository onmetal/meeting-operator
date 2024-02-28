// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const MeetingFinalizer = "onmetal.de/meeting-operator"

func AddFinalizer(ctx context.Context, c client.Client, object client.Object) error {
	if controllerutil.ContainsFinalizer(object, MeetingFinalizer) {
		return nil
	}
	controllerutil.AddFinalizer(object, MeetingFinalizer)
	return c.Update(ctx, object)
}

func RemoveFinalizer(ctx context.Context, c client.Client, object client.Object) error {
	if !controllerutil.ContainsFinalizer(object, MeetingFinalizer) {
		return nil
	}
	controllerutil.RemoveFinalizer(object, MeetingFinalizer)
	return c.Update(ctx, object)
}
