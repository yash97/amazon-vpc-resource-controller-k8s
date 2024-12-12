package ip_latency

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/aws/amazon-vpc-resource-controller-k8s/apis/vpcresources/v1beta1"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/config"
	"github.com/aws/amazon-vpc-resource-controller-k8s/test/framework"
	"github.com/aws/amazon-vpc-resource-controller-k8s/test/framework/manifest"
	sgpWrapper "github.com/aws/amazon-vpc-resource-controller-k8s/test/framework/resource/k8s/sgp"
	"github.com/aws/amazon-vpc-resource-controller-k8s/test/framework/utils"
	verifier "github.com/aws/amazon-vpc-resource-controller-k8s/test/framework/verify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var frameWork *framework.Framework
var verify *verifier.PodVerification
var securityGroupID1 string
var securityGroupID2 string
var ctx context.Context
var err error
var nodeList *v1.NodeList

func TestPodIPLatency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod IP Latency Test Suite")
}

var _ = BeforeSuite(func() {
	By("creating a framework, Security group, verifier")
	frameWork = framework.New(framework.GlobalOptions)
	ctx = context.Background()
	verify = verifier.NewPodVerification(frameWork, ctx)

	securityGroupID1, err = frameWork.EC2Manager.ReCreateSG(utils.ResourceNamePrefix+"sg-1", ctx)
	Expect(err).ToNot(HaveOccurred())

	By("ensuring nodes are ready")
	nodeList, err = frameWork.NodeManager.GetNodeList()
	Expect(err).ToNot(HaveOccurred())
	Expect(nodeList.Items).ShouldNot(BeEmpty(), "No nodes in cluster")
})

var _ = AfterSuite(func() {
	Expect(frameWork.EC2Manager.DeleteSecurityGroup(ctx, securityGroupID1)).To(Succeed())
})

var _ = Describe("Pod IP Assignment Latency Test", func() {
	var (
		securityGroupPolicy *v1beta1.SecurityGroupPolicy

		namespace     string
		sgpLabelKey   string
		sgpLabelValue string
		//podLabelKey    string
		//podLabelValue  string
		securityGroups []string

		err error
	)

	BeforeEach(func() {
		namespace = "per-pod-sg"
		sgpLabelKey = "role"
		sgpLabelValue = "db"
		//podLabelKey = "role"
		//podLabelValue = "db"
		securityGroups = []string{securityGroupID1}
	})

	JustBeforeEach(func() {
		By("creating the namespace if not exist")
		err := frameWork.NSManager.CreateNamespace(ctx, namespace)
		Expect(err).ToNot(HaveOccurred())

		By("creating security group if it does not exists")
		securityGroupPolicy, err = manifest.NewSGPBuilder().
			Namespace(namespace).
			PodMatchLabel(sgpLabelKey, sgpLabelValue).
			SecurityGroup(securityGroups).Build()
		Expect(err).NotTo(HaveOccurred())
		sgpWrapper.CreateSecurityGroupPolicy(frameWork.K8sClient, ctx, securityGroupPolicy)
	})

	JustAfterEach(func() {
		By("deleting security group policy")
		err = frameWork.SGPManager.DeleteAndWaitTillSecurityGroupIsDeleted(ctx, securityGroupPolicy)
		Expect(err).NotTo(HaveOccurred())
		By("deleting namespace")
		err = frameWork.NSManager.DeleteAndWaitTillNamespaceDeleted(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())
	})
	It("should measure IP assignment latency after pod is scheduled on node", func() {
		ctx = context.TODO()
		maxENICapacity := nodeList.Items[0].Status.Allocatable[config.ResourceNamePodENI]
		branchInterfacePerNode, ok := maxENICapacity.AsInt64()
		Expect(ok).To(BeTrue())
		totalPodsPerChurn := 2 * int(branchInterfacePerNode) * len(nodeList.Items)
		fmt.Printf("deploying %d pods per no", totalPodsPerChurn)
		allPodNames := []string{}

		// Performing pod wave
		for wave := 0; wave < 2; wave++ {
			By(fmt.Sprintf("Starting churn iteration %d", wave+1))

			job := manifest.NewLinuxJob().
				Name(fmt.Sprintf("job-%d", wave)).
				Namespace(namespace).
				Parallelism(totalPodsPerChurn). // To accommodate for server Pod
				Container(v1.Container{
					Name:  "test-container",
					Image: "busybox",
					Command: []string{
						"sh",
						"-c",
						"sleep 2",
					},
				}).
				PodLabels(sgpLabelKey, sgpLabelValue).
				PodLabels(fmt.Sprintf("job-churn-%d", wave), "").
				TerminationGracePeriod(30).
				Build()
			_, err = frameWork.JobManager.CreateAndWaitForJobToComplete(ctx, job)
			Expect(err).NotTo(HaveOccurred())
			pods, err := frameWork.PodManager.GetPodsWithLabel(ctx, namespace, fmt.Sprintf("job-churn-%d", wave), "")
			Expect(err).NotTo(HaveOccurred())
			for _, pod := range pods {
				allPodNames = append(allPodNames, pod.Name)
			}

		}

		calculateLatencyFromEvents := func(events []v1.Event) (time.Duration, error) {
			var scheduledTime, allocatedTime time.Time
			for _, event := range events {
				if event.Reason == "Scheduled" {
					scheduledTime = event.LastTimestamp.Time
				} else if event.Reason == "ResourceAllocated" {
					allocatedTime = event.LastTimestamp.Time
				}
			}
			if scheduledTime.IsZero() || allocatedTime.IsZero() {
				return 0, fmt.Errorf("missing required events")
			}
			return allocatedTime.Sub(scheduledTime), nil
		}

		latencies := []time.Duration{}
		pods_missed_events := 0
		time.Sleep(60 * time.Second)
		for _, podName := range allPodNames {
			events, err := frameWork.PodManager.GetPodEvents(ctx, podName, namespace)
			Expect(err).NotTo(HaveOccurred())
			latency, err := calculateLatencyFromEvents(events)
			if err != nil {
				fmt.Println("error finding event for pod ", podName)
				//fmt.Println(events)
				//time.Sleep(120 * time.Second)
				pods_missed_events++
			} else {
				//Expect(err).NotTo(HaveOccurred())
				latencies = append(latencies, latency)
			}
		}
		fmt.Println("pods_missed_event", pods_missed_events)
		if pods_missed_events > 0 {
			time.Sleep(180 * time.Second)
		}
		var totalLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
		}

		avgLatency := totalLatency / time.Duration(len(latencies)-pods_missed_events)
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		fmt.Printf("Average IP assignment latency: %v\n", avgLatency)
		fmt.Printf("Minimum IP latency: %v\n", latencies[0])
		fmt.Printf("Maxmimum IP latency: %v\n", latencies[len(latencies)-1])
		fmt.Printf("Median Ip latency: %v\n", latencies[len(latencies)/2])
	})
})
