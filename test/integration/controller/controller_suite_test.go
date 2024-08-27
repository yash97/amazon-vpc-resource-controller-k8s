package controller_test

import (
	"context"
	"fmt"
	"time"

	//"os"
	"testing"

	"github.com/aws/amazon-vpc-resource-controller-k8s/controllers/custom"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/condition"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/k8s/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var testEnv *envtest.Environment
var cancel context.CancelFunc
var ctx context.Context
var cfg *rest.Config
var err error
var dataStore cache.Indexer
var clientSet *kubernetes.Clientset
var k8sClient client.Client
var r testReconciler
func TestPodController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod Controller tests")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	//ctx, cancel := context.WithCancel(context.TODO())
	By("bootstraping test environment")
	testEnv = &envtest.Environment{
		ControlPlane: envtest.ControlPlane{
			APIServer: &envtest.APIServer{},
			Etcd: &envtest.Etcd{
				Args: []string{
					"--auto-compaction-retention=10s", // Set auto-compaction to 1 minute
				},
			},
		},
		//AttachControlPlaneOutput: true,
	}
	cfg, err = testEnv.Start()
	Expect(cfg).NotTo(BeNil())
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error {
		clientSet, err = kubernetes.NewForConfig(cfg)
		return err
	}, "60s", "1s").Should(Succeed())
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: testEnv.Scheme})
	Expect(err).NotTo(HaveOccurred())
	ctx = context.TODO()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testEnv.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())
	podConverter := &pod.PodConverter{
		K8sResource:     "pods",
		K8sResourceType: &v1.Pod{},
	}
	dataStore = cache.NewIndexer(podConverter.Indexer, pod.NodeNameIndexer())
	conditions := condition.NewControllerConditions(ctrl.Log.WithName("controller conditions"), nil, false)
	
	builder := custom.NewControllerManagedBy(ctx, mgr).Named("test-pod-controller").
		UsingConverter(podConverter).
		WithClientSet(clientSet).
		UsingDataStore(dataStore).
		UsingConditions(conditions).
		WithLogger(ctrl.Log.WithName("custom-controller")).
		UsingConditions(conditions).
		Options(custom.Options{ResyncPeriod: 3 *time.Second})
	_, err = builder.Complete(&r)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())

})

var _ = AfterSuite(func() {
	//cancel()
	By("tearing down the test env")
	Eventually(func() error {
		err := testEnv.Stop()
		return err
	}, "60s", "1s").Should(Succeed())
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Pod controller", func() {
	It("should set up env", func() {
		fmt.Println(testEnv.ControlPlane.Etcd.Configure())
		Expect(true).To(BeTrue())
	})
	It("arithmetic", func() {
		Expect(1 + 1).To(Equal(2))
	})
	It("should print pod info", func() {
		testPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "nginx",
					},
				},
			},
		}
		By("Creating a test pod")
		err := k8sClient.Create(ctx, testPod)
		Expect(err).NotTo(HaveOccurred())

		
		By("Creating a large number of pods")
		for i := 0; i < 1000; i++ {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-pod-%d", i),
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		}
		By("Listing all pods")
		podList := &corev1.PodList{}
		err = k8sClient.List(ctx, podList, &client.ListOptions{Namespace: "default"})
		Expect(err).NotTo(HaveOccurred())
		fmt.Println(len(podList.Items))
		time.Sleep(20 * time.Millisecond)
		By("Continuously updating pods to cause churn")
		go func() {
			for i := 0; i < 1000 && !r.hasEncountered410Error; i++ {
				podName := fmt.Sprintf("test-pod-%d", i%500)
				pod := &corev1.Pod{}
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: podName}, pod)
				if err == nil {
					pod.Labels = map[string]string{"updated": fmt.Sprintf("%d", time.Now().UnixNano())}
					err = k8sClient.Update(ctx, pod)
					if err != nil {
						fmt.Printf("Error updating pod: %v\n", err)
					}
				}
				time.Sleep(3 * time.Millisecond) // Small delay between updates
			}
		}()
		
		By("Waiting for potential 410 Gone error")
		Eventually(func() bool {
			return r.hasEncountered410Error
		}, 3*time.Minute, 1*time.Second).Should(BeTrue())
	})

})

type testReconciler struct{
	hasEncountered410Error bool
}

func (r *testReconciler) Reconcile(request custom.Request) (ctrl.Result, error) {
	if request.DeletedObject != nil {
		// Check if this is a 410 Gone error
		statusError, ok := request.DeletedObject.(*metav1.Status)
		if ok && statusError.Code == 410 {
			r.hasEncountered410Error = true
			fmt.Printf("Encountered 410 Gone error: %v\n", statusError)
		}
	}
	return ctrl.Result{}, nil
}
