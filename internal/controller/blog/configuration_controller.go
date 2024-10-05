/*
Copyright 2024.

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

package blog

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	blogv1 "github.com/deployment-controller/sample-controller.git/api/blog/v1"
)

// ConfigurationReconciler reconciles a Configuration object
type ConfigurationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=blog.core.deploymentcontroller.io,resources=configurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=blog.core.deploymentcontroller.io,resources=configurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=blog.core.deploymentcontroller.io,resources=configurations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Configuration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

func getAppErrorRate(appName string) int {

	return 3
}

func (r *ConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	// Read Configuration CRD

	config := &blogv1.Configuration{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Looping through the Apps from config CRD

	for _, app := range config.Spec.Apps {
		deploymentName := app.Name
		errorThreshold := app.ErrorThreshold
		currentErrorRate := getAppErrorRate(deploymentName)
		l.Info("Monitoring app", "appName", deploymentName, "currentErrorRate", currentErrorRate, "errorThreshold", errorThreshold)

		// Condition for Erorr threshold

		if currentErrorRate > errorThreshold {
			l.Info("Error rate exceeded threshold, restarting app", "appName", deploymentName)
			deploymentNamespacedName := types.NamespacedName{Name: deploymentName, Namespace: req.Namespace}
			deployment := &appsv1.Deployment{}
			if err := r.Get(ctx, deploymentNamespacedName, deployment); err != nil {
				return ctrl.Result{}, err
			}
			// Initialize if no annotations exist
			annotations := deployment.Spec.Template.Annotations
			if annotations == nil {
				annotations = make(map[string]string)
			}
			// Add or update the restart annotation with the current timestamp
			restartTime := time.Now().Format(time.RFC3339)
			annotations["kubectl.kubernetes.io/restartedAt"] = restartTime
			l.Info("Restarting deployment", "deploymentName", deploymentName, "restartTime", restartTime)

			// Update the annotations in the pod template
			deployment.Spec.Template.Annotations = annotations
			if err := r.Update(ctx, deployment); err != nil {
				l.Error(err, "Failed to update the deployment to trigger restart")
				return ctrl.Result{}, err
			}
			l.Info("Deployment restart triggered successfully", "deploymentName", deploymentName)
		}
		/*
			for _, container := range deployment.Spec.Template.Spec.Containers {
				cpuRequest := container.Resources.Requests.Cpu().String()
				memoryRequest := container.Resources.Requests.Memory().String()

				l.Info("Container Resources",
					"Container Name", container.Name,
					"CPU Request", cpuRequest,
					"Memory Request", memoryRequest)
		*/
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&blogv1.Configuration{}).
		Complete(r)
}
