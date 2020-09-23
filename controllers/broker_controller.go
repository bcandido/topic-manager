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
	"k8s.io/apimachinery/pkg/types"
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
	ctx := context.Background()
	log := r.Log.WithValues("broker", req.NamespacedName)

	broker := brokerv1alpha1.Broker{}
	err := r.Client.Get(ctx, req.NamespacedName, &broker)
	log.Info("fetching broker", "broker", req.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("broker not found. Ignoring...")
			return reconcile.Result{}, client.IgnoreNotFound(err) // return and don't requeue
		}
		log.Error(err, "broker not found")
		return reconcile.Result{}, err // requeue the request.
	}

	kafkaConfig := kafka_manager.KafkaConfig{Brokers: broker.ConnectionString()}
	topicController := kafka_manager.New(kafkaConfig)
	if topicController == nil {
		log.Error(err, "error creating kafka client")

		broker.Status.Status = brokerv1alpha1.BrokerOffline
		if err = r.Client.Status().Update(ctx, &broker); err != nil {
			log.Error(err, "error changing broker status to offline", "topic-controller", topicController)
		}
		return reconcile.Result{}, err // requeue the request.
	}

	topic := brokerv1alpha1.Topic{}
	healthCheckTopicName := "topic-manager.broker.health-check." + req.Namespace + "." + req.Name
	key := types.NamespacedName{Name: healthCheckTopicName, Namespace: req.Namespace}
	if err = r.Client.Get(ctx, key, &topic); err != nil && errors.IsNotFound(err) {
		log.Info("broker could not found health check topic", "health-check topic", healthCheckTopicName)
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

		err = r.Client.Create(ctx, healthCheckTopic)
		if err != nil {
			log.Error(err, "error creating health check topic", "health-check topic", healthCheckTopicName)
			return reconcile.Result{}, err // requeue the request.
		}
		log.Info("health check topic created", "health-check topic", healthCheckTopicName)
		return reconcile.Result{Requeue: true}, nil // return and requeue
	}

	if kafkaTopic := topicController.Get(healthCheckTopicName); kafkaTopic == nil {
		broker.Status.Status = brokerv1alpha1.BrokerOffline
		if err = r.Client.Status().Update(ctx, &broker); err != nil {
			log.Error(err, "error changing broker status to offline", "topic-controller", topicController)
		}
		return reconcile.Result{}, err // requeue the request.
	}

	if broker.Status.Status != brokerv1alpha1.BrokerOnline {
		broker.Status.Status = brokerv1alpha1.BrokerOnline
		if err = r.Client.Status().Update(ctx, &broker); err != nil {
			log.Error(err, "error changing broker status to online", "topic-controller", topicController)
		}
	}

	return ctrl.Result{}, nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&brokerv1alpha1.Broker{}).
		Complete(r)
}
