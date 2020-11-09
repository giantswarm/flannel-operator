package flanneld

import (
	"context"
	"strconv"
	"testing"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/k8sclient/k8sclienttest"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

var testConfig = &v1alpha1.FlannelConfig{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "abc12",
		Namespace: "default",
	},
	Spec: v1alpha1.FlannelConfigSpec{
		Cluster: v1alpha1.FlannelConfigSpecCluster{
			ID: "abc12",
		},
	},
}

func Test_GetCurrentState(t *testing.T) {
	testCases := []struct {
		name                 string
		daemonSets           []runtime.Object
		expectedCancellation bool
	}{
		{
			name: "case 0: network matches and is ready should not cancel",
			daemonSets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "flannel-network",
						},
						Name:      "flannel-network",
						Namespace: "flannel-network-abc12",
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1,
						NumberReady:            1,
					},
				},
			},
			expectedCancellation: false,
		},
		{
			name: "case 1: network matches and is not ready should not cancel",
			daemonSets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "flannel-network",
						},
						Name:      "flannel-network",
						Namespace: "flannel-network-abc12",
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1,
						NumberReady:            0,
					},
				},
			},
			expectedCancellation: false,
		},
		{
			name: "case 2: a different network which is ready should not cancel",
			daemonSets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "flannel-network",
						},
						Name:      "flannel-network",
						Namespace: "flannel-network-def34",
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1,
						NumberReady:            1,
					},
				},
			},
			expectedCancellation: false,
		},
		{
			name: "case 3: a different network which is not ready should cancel",
			daemonSets: []runtime.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "flannel-network",
						},
						Name:      "flannel-network",
						Namespace: "flannel-network-def34",
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1,
						NumberReady:            0,
					},
				},
			},
			expectedCancellation: true,
		},
		{
			name:                 "case 4: no networks",
			expectedCancellation: false,
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			fakeK8sClient, err := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
				K8sClient: fake.NewSimpleClientset(tc.daemonSets...),
			})
			if err != nil {
				t.Fatal(err)
			}

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatal(err)
			}

			r := Resource{
				k8sClient: fakeK8sClient.K8sClient(),
				logger:    logger,
			}
			ctx := reconciliationcanceledcontext.NewContext(context.Background(), make(chan struct{}))
			_, err = r.GetCurrentState(ctx, testConfig)
			if !cmp.Equal(err, nil) {
				t.Fatalf("\n\n%s\n", cmp.Diff(err, nil))
			}

			cancelled := reconciliationcanceledcontext.IsCanceled(ctx)
			if !cmp.Equal(cancelled, tc.expectedCancellation) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expectedCancellation, cancelled))
			}
		})
	}
}
