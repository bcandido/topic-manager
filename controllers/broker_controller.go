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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	brokerv1alpha1 "github.com/bcandido/topic-manager/api/v1alpha1"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=broker.bcandido.com,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=broker.bcandido.com,resources=brokers/status,verbs=get;update;patch

func (r *BrokerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("broker", req.NamespacedName)

	broker := &brokerv1alpha1.Broker{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, broker)
	r.Log.Info("fetched broker", "Broker.Spec", broker.Spec)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object
		// requeue the request.
		return reconcile.Result{}, err
	}

	kafkaConfig := kafka_manager.KafkaConfig{Brokers: broker.ConnectionString()}
	topicController := kafka_manager.New(kafkaConfig)
	if topicController == nil {
		// Error creating kafka client
		// requeue the request.
		return reconcile.Result{}, err
	}

	healthCheckTopicName := "topic-manager.broker.health-check." + req.Namespace + "." + req.Name
	kafkaTopic := topicController.Get(healthCheckTopicName)
	if kafkaTopic != nil {
		// TODO: set status to online

		// Topic with same name on broker already exists
		// Return and don't requeue
		return reconcile.Result{}, nil
	}

	healthCheckTopic := &brokerv1alpha1.Topic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      healthCheckTopicName,
			Namespace: req.Namespace,
		},
		Spec: brokerv1alpha1.TopicSpec{
			Name:   healthCheckTopicName,
			Broker: broker.Name,
			Configuration: brokerv1alpha1.TopicConfiguration{
				Partitions:        1,
				ReplicationFactor: 1,
			},
		},
	}

	if err := controllerutil.SetControllerReference(broker, healthCheckTopic, r.Scheme); err != nil {
		// Error setting controller reference - requeue the request.
		// requeue the request.
		return reconcile.Result{}, err
	}

	err = r.Client.Create(context.TODO(), healthCheckTopic)
	if err != nil {
		// Error creating health check topic
		// requeue the request.
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&brokerv1alpha1.Broker{}).
		Complete(r)
}
