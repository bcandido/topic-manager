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
	kafka_manager "github.com/bcandido/topic-controller"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	brokerv1alpha1 "github.com/bcandido/topic-manager/api/v1alpha1"
)

// TopicReconciler reconciles a Topic object
type TopicReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=broker.bcandido.com,resources=topics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=broker.bcandido.com,resources=topics/status,verbs=get;update;patch

func (r *TopicReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("topic", req.NamespacedName)

	log.Info("Reconciling Topic")

	topic := brokerv1alpha1.Topic{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, &topic)
	r.Log.Info("fetched requested topic", "Topic.Spec", topic.Spec)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	broker := brokerv1alpha1.Broker{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: topic.Spec.Broker, Namespace: topic.Namespace}, &broker)
	r.Log.Info("fetched broker reference", "Broker.Spec", broker.Spec)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("topic not found. Ignoring...")
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
		if err = r.setStatus(topic, brokerv1alpha1.TopicStatusFailure); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	if broker.Status.Status == brokerv1alpha1.BrokerOffline {
		log.Info("broker offline. retrying later", "broker", broker.Spec.Name)
		if err = r.setStatus(topic, brokerv1alpha1.TopicStatusCreating); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{RequeueAfter: 5 * second}, nil
	}

	topicController, err := buildTopicController(&broker)
	if err != nil {
		if err = r.setStatus(topic, brokerv1alpha1.TopicStatusCreating); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	kafkaTopic := topicController.Get(topic.Spec.Name)
	if kafkaTopic == nil {
		err = topicController.Create(kafka_manager.Topic{
			Name:              topic.Spec.Name,
			Partitions:        topic.Spec.Configuration.Partitions,
			ReplicationFactor: topic.Spec.Configuration.ReplicationFactor,
		})
		if err != nil {
			r.Log.Info("Error creating topic", "topic", topic.Spec)
			if err = r.setStatus(topic, brokerv1alpha1.TopicStatusFailure); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
	}

	if err = r.setStatus(topic, brokerv1alpha1.TopicStatusCreated); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&brokerv1alpha1.Topic{}).
		Complete(r)
}

func (r *TopicReconciler) setStatus(topic brokerv1alpha1.Topic, status brokerv1alpha1.TopicStatusValue) error {
	log := r.Log.WithValues("topic", topic.ObjectMeta.Name, "namespace", topic.ObjectMeta.Namespace)
	log.Info("updating topic status", "status", status)
	if status == topic.Status.Status {
		log.Info("topic is already with correctly status", "status", status)
		return nil
	}

	topic.Status.Status = status
	if err := r.Client.Status().Update(context.TODO(), &topic); err != nil {
		return err
	}
	log.Info("topic status updated to", "status", status)
	return nil
}
