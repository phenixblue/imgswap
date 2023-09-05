/*
Copyright 2023 The Webroot, Inc.

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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mapsv1alpha1 "twr.dev/imgswap/api/v1alpha1"
	"twr.dev/imgswap/internal/mapstore"
)

// SwapMapReconciler reconciles a SwapMap object
type SwapMapReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	MapStore *mapstore.MapStore
}

//+kubebuilder:rbac:groups=maps.k8s.imgswap.io,resources=swapmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=maps.k8s.imgswap.io,resources=swapmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=maps.k8s.imgswap.io,resources=swapmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SwapMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *SwapMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	var swapMap mapsv1alpha1.SwapMap

	err := r.Client.Get(ctx, req.NamespacedName, &swapMap)
	if err != nil {
		log.Log.Error(err, "unable to fetch SwapMap")
		return ctrl.Result{}, err
	}

	log.Log.Info("Got SwapMap", "name", swapMap.Name)

	for _, mapSpec := range swapMap.Spec.Maps {
		mapKey, err := mapstore.GetMapKey(mapSpec)
		fmt.Printf("Map: %v", mapSpec)
		if err != nil {
			log.Log.Error(err, "unable to get map key")
			return ctrl.Result{}, err
		}
		r.MapStore.AddOrUpdate(mapKey, &mapSpec)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwapMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mapsv1alpha1.SwapMap{}).
		Complete(r)
}
