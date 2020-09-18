/*


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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-logr/logr"
	errs "github.com/pkg/errors"

	cloudhsmv1alpha1 "github.com/hhamalai/cloudhsm-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CloudHSMReconciler reconciles a CloudHSM object
type CloudHSMReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	ctx *Context
}

var RequeueAfter = 60 * time.Second

// +kubebuilder:rbac:groups=cloudhsm.hhamalai.net,resources=cloudhsms;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudhsm.hhamalai.net,resources=cloudhsms/status,verbs=get;update;patch

func (r *CloudHSMReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	log := r.Log.WithValues("cloudhsm", req.NamespacedName)

	// Fetch the CloudHSM instance
	instance := &cloudhsmv1alpha1.CloudHSM{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Info("CloudHSM operator reconciling", "ClusterId", instance.Spec.ClusterId)

	// Define a new ConfigMap object
	cm, err := r.newCMForCR(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Set CloudHSM instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, cm, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this CM already exists
	found := &corev1.ConfigMap{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Client.Create(context.TODO(), cm)
		if err != nil {
			return reconcile.Result{}, err
		}

		// CM created successfully - don't requeue
		return reconcile.Result{RequeueAfter: RequeueAfter}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Updating ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
	err = r.Client.Update(context.TODO(), cm)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: RequeueAfter}, nil
}

// returns a configmap with CloudHSM description with the same name/namespace as the cr
func (r *CloudHSMReconciler) newCMForCR(cr *cloudhsmv1alpha1.CloudHSM) (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app": cr.Name,
	}
	if r.ctx == nil {
		r.ctx = newContext(nil)
	}

	ref := cr.Spec.ClusterId
	data, err := r.ctx.GetHSMIPs(ref)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get json secret as map")
	}
	hsmIps := make(map[string]string)
	for i := range data {
		hsmIps[fmt.Sprintf("hsm_ip.%d", i)] = aws.StringValue(data[i])
	}
	hsmIps["hsm_ips"] = strings.Join(aws.StringValueSlice(data), ",")
	if len(data) > 0 {
		hsmIps["hsm_first_ip"] = aws.StringValue(data[0])
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: hsmIps,
	}, nil
}

func (r *CloudHSMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudhsmv1alpha1.CloudHSM{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
