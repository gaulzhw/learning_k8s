package clientgo

import (
	"log"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func StartInformer() error {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	// 初始化 rest.Config 对象
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// 创建 Clientset 对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// 初始化 informer factory
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)
	// 对 Deployment 监听
	deployInformer := informerFactory.Apps().V1().Deployments()
	// 创建 Informer（相当于注册到工厂中去，这样下面启动的时候就会去 List & Watch 对应的资源）
	informer := deployInformer.Informer()
	// 创建 Lister
	deployLister := deployInformer.Lister()
	// 注册事件处理程序
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onAdd,
		UpdateFunc: onUpdate,
		DeleteFunc: onDelete,
	})

	stopper := make(chan struct{})
	defer close(stopper)

	// 启动 informer，List & Watch
	informerFactory.Start(stopper)
	// 等待所有启动的 Informer 的缓存被同步
	informerFactory.WaitForCacheSync(stopper)
	log.Println("synced")

	// 从本地缓存中获取 default 中的所有 deployment 列表
	deployments, err := deployLister.Deployments("default").List(labels.Everything())
	if err != nil {
		return err
	}
	for idx, deploy := range deployments {
		log.Printf("%d -> %s\n", idx+1, deploy.Name)
	}
	<-stopper

	return nil
}

func onAdd(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	log.Println("add a deployment:", deploy.Name)
}

func onUpdate(old, new interface{}) {
	oldDeploy := old.(*v1.Deployment)
	newDeploy := new.(*v1.Deployment)
	log.Println("update deployment:", oldDeploy.Name, newDeploy.Name)
}

func onDelete(obj interface{}) {
	deploy := obj.(*v1.Deployment)
	log.Println("delete a deployment:", deploy.Name)
}

func StartInformerWithIndex() error {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)
	podInformer := informerFactory.Core().V1().Pods().Informer()
	podInformer.GetIndexer().AddIndexers(cache.Indexers{
		"nodeName": indexByPodNodeName,
	})

	stopChan := make(chan struct{})
	defer close(stopChan)

	informerFactory.Start(stopChan)
	informerFactory.WaitForCacheSync(stopChan)

	podList, err := podInformer.GetIndexer().ByIndex("nodeName", "hub-control-plane")
	if err != nil {
		return err
	}

	for _, pod := range podList {
		podObj := pod.(*corev1.Pod)
		log.Println(podObj)
	}
	return nil
}

func indexByPodNodeName(obj interface{}) ([]string, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return []string{}, nil
	}

	if len(pod.Spec.NodeName) == 0 || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return []string{}, nil
	}

	return []string{pod.Spec.NodeName}, nil
}
