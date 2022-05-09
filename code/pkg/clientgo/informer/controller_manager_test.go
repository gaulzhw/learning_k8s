package informer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wI2L/jsondiff"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type PodController struct {
	mgr ctrl.Manager
}

func (c *PodController) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "spec.nodeName", func(obj client.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return []string{}
		}
		if len(pod.Spec.NodeName) == 0 || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return []string{}
		}
		return []string{pod.Spec.NodeName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return true
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return true
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		}).
		Complete(c)
}

func (c *PodController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	pod := &corev1.Pod{}
	if err := c.mgr.GetClient().Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, err
	}
	fmt.Printf("pod: %v", pod)
	return ctrl.Result{}, nil
}

func TestController(t *testing.T) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		ClientDisableCacheFor: []client.Object{
			&corev1.Pod{}, // 配置disable cache的object，会watch资源，client不会走cache
		},
		NewCache: cache.BuilderWithOptions(cache.Options{
			DefaultSelector: cache.ObjectSelector{Field: fields.OneTermEqualSelector("metadata.namespace", "default")},
			SelectorsByObject: map[client.Object]cache.ObjectSelector{
				&corev1.Pod{}: {Field: fields.OneTermEqualSelector("metadata.namespace", "kube-system")},
			},
		}),
	})
	assert.NoError(t, err)

	if err := (&PodController{
		mgr: mgr,
	}).SetupWithManager(mgr, controller.Options{
		MaxConcurrentReconciles: 1,
	}); err != nil {
		assert.NoError(t, err)
	}

	ctx := ctrl.SetupSignalHandler()
	if err := mgr.Start(ctx); err != nil {
		assert.NoError(t, err)
	}
}

func TestPatch(t *testing.T) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		//Namespace:          osArgs.WatchNamespace,
		//SyncPeriod:         &osArgs.SyncPeriod,
		//MetricsBindAddress: osArgs.MetricsAddr,
	})
	assert.NoError(t, err)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
		},
	}
	err = mgr.GetAPIReader().Get(context.TODO(), client.ObjectKeyFromObject(deploy), deploy)
	assert.NoError(t, err)

	newDeploy := deploy.DeepCopy()
	newDeploy.ObjectMeta.Labels["test1"] = "difftest1"

	oldObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deploy)
	newObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(newDeploy)

	spec, _, err := unstructured.NestedFieldNoCopy(oldObj, "spec")
	t.Logf("spec: %v", spec)

	// 比较新旧deploy的不同，返回不同的bytes
	patch, err := jsondiff.Compare(oldObj, newObj)
	assert.NoError(t, err)

	// 打patch，patchBytes就是我们需要的了
	patchBytes, err := json.Marshal(patch)
	assert.NoError(t, err)

	err = mgr.GetClient().Patch(context.TODO(), deploy, client.RawPatch(types.JSONPatchType, patchBytes))
	assert.NoError(t, err)
}
