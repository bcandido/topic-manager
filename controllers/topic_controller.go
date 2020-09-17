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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"

	kafka_manager "github.com/bcandido/topic-controller"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Topic")

	topic := &brokerv1alpha1.Topic{}
	key := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	err := r.Client.Get(context.TODO(), key, topic)
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

	broker := &brokerv1alpha1.Broker{}
	key = types.NamespacedName{Name: topic.Spec.Broker, Namespace: topic.Namespace}
	err = r.Client.Get(context.TODO(), key, broker)
	r.Log.Info("fetched broker reference", "Broker.Spec", broker.Spec)
	if err != nil && errors.IsNotFound(err) {

		// update status to failure
		topic.Status.Value = brokerv1alpha1.TopicStatusFailure
		err = r.Client.Status().Update(context.Background(), topic)
		if err != nil {
			r.Log.Error(err, "unable to update topic status")
			return reconcile.Result{}, err //requeue
		}

		// Broker reference could no be found, then topic cannot be created on broker
		// Return and don't requeue
		return ctrl.Result{Requeue: false}, err
	}

	kafkaConfig := kafka_manager.KafkaConfig{Brokers: getBrokerConnectionString(broker)}
	topicController, err := kafka_manager.New(kafkaConfig)
	if err != nil {
		// update status to failure
		topic.Status.Value = brokerv1alpha1.TopicStatusFailure
		err = r.Client.Status().Update(context.Background(), topic)
		if err != nil {
			r.Log.Error(err, "unable to update topic status")
			return reconcile.Result{}, err //requeue
		}
	}

	err = topicController.Create(kafka_manager.Topic{
		Name:              topic.Spec.Name,
		Partitions:        topic.Spec.Configuration.Partitions,
		ReplicationFactor: topic.Spec.Configuration.ReplicationFactor,
	})
	if err != nil {
		// update status to failure
		topic.Status.Value = brokerv1alpha1.TopicStatusFailure
		err = r.Client.Status().Update(context.Background(), topic)
		if err != nil {
			r.Log.Error(err, "unable to update topic status")
			return reconcile.Result{}, err //requeue
		}

		r.Log.Info("Error creating topic", "topic", topic.Spec)
		return reconcile.Result{}, err
	}

	topic.Status.Value = brokerv1alpha1.TopicStatusCreated
	err = r.Client.Status().Update(context.Background(), topic)
	if err != nil {
		r.Log.Error(err, "unable to update topic status")
		return reconcile.Result{}, err //requeue
	}

	// created successfully
	return ctrl.Result{}, nil
}

func getBrokerConnectionString(broker *brokerv1alpha1.Broker) string {
	return strings.Join(broker.Spec.Configuration.BootstrapServers, ",")
}

func shouldCreateTopic(err error) bool {
	return err != nil && errors.IsNotFound(err)
}

func (r *TopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&brokerv1alpha1.Topic{}).
		Complete(r)
}
