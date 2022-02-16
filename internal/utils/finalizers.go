// /*
// Copyright (c) 2021 T-Systems International GmbH, SAP SE or an SAP affiliate company. All right reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
