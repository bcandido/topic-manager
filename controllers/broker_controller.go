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
	kafkamanager "github.com/bcandido/topic-controller"
	brokerv1alpha1 "github.com/bcandido/topic-manager/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	healthCheckTopicPrefix = "topic-manager.broker.health-check"
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
	log := r.Log.WithValues("broker", req.NamespacedName)

	log.Info("Reconciling Broker")

	broker := brokerv1alpha1.Broker{}
	if err := r.fetchBrokerFromRequest(req, &broker); err != nil {
		return reconcile.Result{}, err
	}

	topicController, err := buildTopicController(&broker)
	if err != nil {
		if err = r.setStatusOffline(broker); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	log.Info("checking broker connectivity")
	healthCheckTopicName := r.getHealthCheckTopicName(req)
	kafkaTopic, err := r.checkingConnectivity(topicController, healthCheckTopicName)
	if err != nil {
		if err = r.setStatusOffline(broker); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, err
	}
	log.Info("broker connectivity health")

	healthCheckTopic := r.buildHealthCheckTopic(req, broker, kafkaTopic)
	err = r.createTopicResource(req, healthCheckTopic)
	if err != nil {
		return reconcile.Result{}, err
	}

	if broker.Status.Status != brokerv1alpha1.BrokerOnline {
		if err = r.setStatusOnline(broker); err != nil {
			return reconcile.Result{}, err
		}
	}

	log.Info("broker successfully reconciled")
	return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Minute}, nil
}

func (r *BrokerReconciler) getHealthCheckTopicName(req ctrl.Request) string {
	return fmt.Sprintf("%v.%v.%v", healthCheckTopicPrefix, req.Namespace, req.Name)
}

func (r *BrokerReconciler) fetchBrokerFromRequest(req ctrl.Request, broker *brokerv1alpha1.Broker) error {
	log := r.Log.WithValues("broker", req.NamespacedName)
	err := r.Client.Get(context.TODO(), req.NamespacedName, broker)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("broker not found. Ignoring...")
			return client.IgnoreNotFound(err)
		}
		log.Error(err, "broker not found")
		return err
	}
	return nil
}

func (r *BrokerReconciler) buildHealthCheckTopic(req ctrl.Request, broker brokerv1alpha1.Broker, kafkaTopic *kafkamanager.Topic) *brokerv1alpha1.Topic {
	return &brokerv1alpha1.Topic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaTopic.Name,
			Namespace: req.Namespace,
		},
		Spec: brokerv1alpha1.TopicSpec{
			Broker: broker.Name,
			Configuration: brokerv1alpha1.TopicConfiguration{
				Partitions:        kafkaTopic.Partitions,
				ReplicationFactor: kafkaTopic.ReplicationFactor,
			},
		},
	}
}

func (r *BrokerReconciler) checkingConnectivity(topicController kafkamanager.TopicControllerAPI, topicName string) (*kafkamanager.Topic, error) {
	var err error = nil
	kafkaTopic := topicController.Get(topicName)
	if kafkaTopic == nil {
		err = errors.NewServiceUnavailable("broker connectivity fail with topic " + topicName)
	}
	return kafkaTopic, err
}

func (r *BrokerReconciler) createTopicResource(req ctrl.Request, topic *brokerv1alpha1.Topic) error {
	log := r.Log.WithValues("broker", req.NamespacedName)

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: topic.Name, Namespace: req.Namespace}, topic)
	if err != nil && errors.IsNotFound(err) {
		log.Info("creating topic", "topic", topic.Name)
		err = r.Client.Create(context.TODO(), topic)
		if err != nil {
			log.Error(err, "error creating topic", "topic", topic.Name)
			return err
		}
		log.Info("topic created", "topic", topic.Name)
		return nil
	}
	return nil
}

func (r *BrokerReconciler) setStatusOffline(broker brokerv1alpha1.Broker) error {
	return r.setStatus(broker, brokerv1alpha1.BrokerOffline)
}

func (r *BrokerReconciler) setStatusOnline(broker brokerv1alpha1.Broker) error {
	return r.setStatus(broker, brokerv1alpha1.BrokerOnline)
}

func (r *BrokerReconciler) setStatus(broker brokerv1alpha1.Broker, status brokerv1alpha1.BrokerStatusValue) error {
	log := r.Log.WithValues("broker", broker.ObjectMeta.Name, "namespace", broker.ObjectMeta.Namespace)
	log.Info("updating broker status", "status", status)
	if status == broker.Status.Status {
		log.Info("broker is already with correctly status", "status", status)
		return nil
	}

	broker.Status.Status = status
	if err := r.Client.Status().Update(context.TODO(), &broker); err != nil {
		return err
	}
	log.Info("broker status updated to", "status", status)
	return nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&brokerv1alpha1.Broker{}).
		Complete(r)
}
