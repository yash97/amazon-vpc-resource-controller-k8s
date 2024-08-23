package controller_test

import (
	"context"
	"fmt"
	//"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var testEnv *envtest.Environment
var cancel context.CancelFunc
var ctx context.Context
var k8sClient *rest.Config
var err error
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
			Etcd:      &envtest.Etcd{},
		},
		AttachControlPlaneOutput: true,
	}
	testEnv.ControlPlane.Etcd.Configure().Append("auto-compaction-retention", "1m")
	k8sClient, err = testEnv.Start()
	testEnv.ControlPlane.Etcd.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	//cancel()
	By("tearing down the test env")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Pod controller", func(){
	It("should set up env",func(){
		fmt.Print(testEnv.ControlPlane.Etcd.Configure())
		Expect(true).To(BeTrue())

	})
})
