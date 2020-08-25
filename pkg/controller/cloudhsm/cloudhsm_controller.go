package cloudhsm

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"strings"
	"time"

	cloudhsmv1alpha1 "github.com/hhamalai/cloudhsm-operator/pkg/apis/cloudhsm/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	errs "github.com/pkg/errors"
)

var REQUEUE_AFTER = 60 * time.Second
var log = logf.Log.WithName("controller_cloudhsm")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CloudHSM Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCloudHSM{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cloudhsm-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CloudHSM
	err = c.Watch(&source.Kind{Type: &cloudhsmv1alpha1.CloudHSM{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CloudHSM
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cloudhsmv1alpha1.CloudHSM{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCloudHSM implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCloudHSM{}

// ReconcileCloudHSM reconciles a CloudHSM object
type ReconcileCloudHSM struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	ctx *Context
}

// Reconcile reads that state of the cluster for a CloudHSM object and makes changes based on the state read
// and what is in the CloudHSM.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCloudHSM) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CloudHSM")

	// Fetch the CloudHSM instance
	instance := &cloudhsmv1alpha1.CloudHSM{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	reqLogger.Info("cloudhsm operator reconciling", "ClusterId", instance.Spec.ClusterId)

	// Define a new ConfigMap object
	cm, err := r.newCMForCR(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Set CloudHSM instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this CM already exists
	found := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			return reconcile.Result{}, err
		}

		// CM created successfully - don't requeue
		return reconcile.Result{RequeueAfter: REQUEUE_AFTER}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Updating ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
	err = r.client.Update(context.TODO(), cm)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{RequeueAfter: REQUEUE_AFTER}, nil
}

// returns a configmap with CloudHSM description with the same name/namespace as the cr
func (r *ReconcileCloudHSM) newCMForCR(cr *cloudhsmv1alpha1.CloudHSM) (*corev1.ConfigMap, error) {
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
	hsm_ips := make(map[string]string)
	for i := range data {
		hsm_ips[fmt.Sprintf("hsm_ip.%d", i)] = aws.StringValue(data[i])
	}
	hsm_ips["hsm_ips"] = strings.Join(aws.StringValueSlice(data), ",")
	if len(data) > 0 {
		hsm_ips["hsm_first_ip"] = aws.StringValue(data[0])
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: hsm_ips,
	}, nil
}
