package k8s

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/pkg/api/v1"
	"k8s.io/api/core/v1"
	//v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	//podUtil "gw.com.cn/dzhyun/yunconsole2.git/k8s/util/pod"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"strings"
	"time"
)

var kubeSystemNamespace = []string{"kube-system", "kube-public"}

// PodLifeCycle record service's pod life cycle

type PodController struct {
	Clientset     *kubernetes.Clientset
	client        K8sClient
	PodController cache.Controller
	PodLister     cache.Indexer
	Queue         workqueue.RateLimitingInterface
}

// NewPodController return PodController
func NewPodController(client *kubernetes.Clientset, resyncPeriod time.Duration) *PodController {
	pc := &PodController{}
	podListerWather := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "pods", v1.NamespaceAll, fields.Everything())
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	resourceEventHandle := cache.ResourceEventHandlerFuncs{AddFunc: pc.OnAdd, DeleteFunc: pc.OnDelete}
	podIndex, podInformer := cache.NewIndexerInformer(podListerWather, &v1.Pod{}, resyncPeriod, resourceEventHandle, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	pc.Clientset = client
	pc.PodController = podInformer
	pc.PodLister = podIndex
	pc.Queue = queue
	return pc
}

// OnAdd add event call func
func (pc *PodController) OnAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	obj, exist, err := pc.PodLister.GetByKey(key)
	if err != nil {
		return
	}
	if !exist {
		return
	}

	if p, ok := obj.(*v1.Pod); ok {
		if StringNotIn(kubeSystemNamespace, p.Namespace) {
			item := db.PodLifeCycle{}
			item.Namespace = p.Namespace
			item.PodName = p.Name
			item.CreateAt = p.CreationTimestamp.Time
			item.UpdateAt = time.Now()
			item.Action = "add"
			item.Status = string(p.Status.Phase)
			item.Insert()
		}
	}
}

// OnUpdate update event call func
func (pc *PodController) OnUpdate(old, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		pc.Queue.Add(key)
	}

	obj, exist, err := pc.PodLister.GetByKey(key)
	if err != nil {
		return
	}
	action := "update"
	if !exist {
		action = "stopped"
	}

	if p, ok := obj.(*v1.Pod); ok {
		if StringNotIn(kubeSystemNamespace, p.Namespace) {
			item := db.PodLifeCycle{}
			item.Namespace = p.Namespace
			item.PodName = p.Name
			item.UpdateAt = time.Now()
			item.Status = string(p.Status.Phase)
			item.Action = action
			item.Update()
		}
	}

}

// OnDelete delete event call func
func (pc *PodController) OnDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		pc.Queue.Add(key)
	}
	obj, exist, err := pc.PodLister.GetByKey(key)
	if err != nil {
		return
	}
	item := db.PodLifeCycle{}
	action := "delete"
	item.UpdateAt = time.Now()
	if !exist {

	}
	if p, ok := obj.(*v1.Pod); ok {
		if StringNotIn(kubeSystemNamespace, p.Namespace) {

			item.Namespace = p.Namespace
			item.PodName = p.Name
			item.Status = string(p.Status.Phase)
			item.Action = action

		}
	} else {
		values := strings.SplitN(key, "/", 2)
		if len(values) == 2 {
			item.Namespace = values[0]
			item.PodName = values[1]
		}
		item.Action = "stopped"
		item.Status = "deleted"
	}
	err = item.Update()

	if err != nil {

	}
}

// Run start pod controller
func (pc *PodController) Run(stopCh chan struct{}) {
	pc.PodController.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, pc.PodController.HasSynced) {
		//runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	wait.Until(pc.worker, time.Second, stopCh)

	<-stopCh

}

func (pc *PodController) worker() {

	for pc.processNextItem() {
	}
}

func (pc *PodController) processNextItem() bool {
	key, shutDown := pc.Queue.Get()
	if shutDown {
		return false
	}
	defer func() {
		pc.Queue.Done(key)
	}()

	return true
}

// IntNotIn assert ojb in the des int array or not
func IntNotIn(des []int, obj int) bool {
	for _, item := range des {
		if item == obj {
			return true
		}
	}
	return false
}

// StringNotIn assert ojb in the des  string array or not
func StringNotIn(des []string, obj string) bool {
	for _, item := range des {
		if item == obj {
			return false
		}
	}
	return true
}
