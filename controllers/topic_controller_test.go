package controllers

import (
	brokerv1alpha1 "github.com/bcandido/topic-manager/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/klogr"
	"reflect"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

type fields struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}
type args struct {
	req controllerruntime.Request
}

type TestCase struct {
	name    string
	fields  fields
	args    args
	want    controllerruntime.Result
	wantErr bool
}

func TestTopicReconciler_Reconcile(t *testing.T) {
	var (
		name              = "test_topic"
		namespace         = "test-namespace"
		partitions        = 15
		replicationFactor = 3
	)

	topic := &brokerv1alpha1.Topic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: brokerv1alpha1.TopicSpec{
			Name:   name,
			Broker: "localhost:9092",
			Configuration: brokerv1alpha1.TopicConfiguration{
				Partitions:        partitions,
				ReplicationFactor: replicationFactor,
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(brokerv1alpha1.GroupVersion, topic)
	cl := fake.NewFakeClientWithScheme(s, topic)

	fields := fields{
		Client: cl,
		Log: klogr.New(),
		Scheme: s,
	}

	args := args{req: reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}}

	want := controllerruntime.Result{Requeue: false}

	tests := []TestCase{
		{"test something", fields, args, want, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TopicReconciler{
				Client: tt.fields.Client,
				Log:    tt.fields.Log,
				Scheme: tt.fields.Scheme,
			}
			got, err := r.Reconcile(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
