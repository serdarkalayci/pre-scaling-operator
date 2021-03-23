package controllers

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Deployment Watch Controller", func() {

	const timeout = time.Second * 25
	const interval = time.Millisecond * 200

	var casenumber = 1

	var deployment v1.Deployment
	//var clusterscalingstate *v1alpha1.ClusterScalingState

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}

	BeforeEach(func() {

	})

	AfterEach(func() {
		// Tear down the deployment
		Expect(k8sClient.Delete(context.Background(), &deployment)).Should(Succeed())

		casenumber = casenumber + 1
		time.Sleep(time.Second * 1)
	})

	Context("Deployment change Test", func() {
		When("A deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(annotationchange bool, replicachange bool, optinOld bool, optinNew bool, expectedReplicas int) {
				key.Name = "case" + strconv.Itoa(casenumber)

				deployment = CreateDeployment(key, optinOld, casenumber)
				Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())
				time.Sleep(time.Second * 2)

				// Get the deployment (created by the Before() step to edit it according to the test)
				fetched := v1.Deployment{}
				Eventually(func() v1.Deployment {
					k8sClient.Get(context.Background(), key, &fetched)
					return fetched
				}, timeout, interval).Should(Not(BeNil()))

				if annotationchange {
					fetched = ChangeAnnotation(fetched)
				}

				if replicachange {
					fetched = ChangeReplicas(fetched)
				}

				// Enable/Disable optin
				fetched = ChangeOptIn(fetched, optinNew)

				// Update with the new changes
				By("Then a deployment is updated")
				Expect(k8sClient.Update(context.Background(), &fetched)).Should(Succeed())

				time.Sleep(time.Second * 2)

				var replicas32 int32 = int32(expectedReplicas)

				Eventually(func() int32 {
					k8sClient.Get(context.Background(), key, &fetched)
					return *fetched.Spec.Replicas
				}, timeout, interval).Should(Equal(replicas32))
			},

				// Default Replica Count from test if oldoptin = true: 3
				// Default Replica Count from test if oldoptin = false: 1
				// Default fallback annotation count: 2
				// bau Annoation (if changed) will change to :4
				// Replica change (that needs to be rectified): 5
				// Structure:  ("Description of the case" , annotationchange, replicachange, oldoptin, newoptin, expectedReplicas)
				table.Entry("CASE 1  | Should scale to 4 | Annotation has changed. ", true, true, true, true, 4),
				table.Entry("CASE 2  | Should scale to 2 | Deployment has been disabled, fallback to default annotation", true, true, true, false, 2),
				table.Entry("CASE 3  | Should scale to 4 | Deployment opted in and annotation changed", true, true, false, true, 4),
				table.Entry("CASE 4  | Should be at 5    | Deployment opted out. Will go to modified replica count (5)", true, true, false, false, 5),
				table.Entry("CASE 5  | Should scale to 4 | Annotation has been modified.", true, false, true, true, 4),
				table.Entry("CASE 6  | Should scale to 2 | Deployment has been disabled fallback to default annotation", true, false, true, false, 2),
				table.Entry("CASE 7  | Should scale to 4 | Deployment opted in and annotation changed.", true, false, false, true, 4),
				table.Entry("CASE 8  | Should be at 1    | Deployment opted out", true, false, false, false, 1),
				table.Entry("CASE 9  | Should scale to 3 | Replica count has been modified. Rectify back to 'bau'", false, true, true, true, 3),
				table.Entry("CASE 10 | Should scale to 2 | Deployment has been disabled fallback to default annotation", false, true, true, false, 2),
				table.Entry("CASE 11 | Should scale to 3 | Deployment opted in. Scale to 'bau'", false, true, false, true, 3),
				table.Entry("CASE 12 | Should be at 5    | Deployment opted out. Will go to modified replica count (5)", false, true, false, false, 5),
				table.Entry("CASE 13 | Stay at Bau  	 | Something else on the deployment changed. Don't do anything", false, false, true, true, 3),
				table.Entry("CASE 14 | Should scale to 2 | Deployment has been disabled fallback to default annotation", false, false, true, false, 2),
				table.Entry("CASE 15 | Should scale to 3 | Deployment opted in. Scale to 'bau'", false, false, false, true, 3),
				table.Entry("CASE 16 | Should be at 1    | Nothing happend", false, false, false, false, 1),
			)
		})
	})

})

func ChangeAnnotation(deployment v1.Deployment) v1.Deployment {
	annotations := map[string]string{
		"scaler/state-bau-replicas":     "4", // That reflects the annotation change and will change replica # to 4
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	deployment.Annotations = annotations

	return deployment
}

func ChangeOptIn(deployment v1.Deployment, optIn bool) v1.Deployment {

	labels := map[string]string{
		"app":           "random-generator-1",
		"scaler/opt-in": strconv.FormatBool(optIn),
	}

	deployment.Labels = labels
	return deployment
}

func ChangeReplicas(deployment v1.Deployment) v1.Deployment {
	var replicas int32 = 5

	spec2 := deployment.Spec
	spec2.Replicas = &replicas

	deployment.Spec = spec2
	return deployment
}

func CreateDeployment(deploymentInfo types.NamespacedName, optInOld bool, casenumber int) v1.Deployment {
	var replicaCount int32
	if optInOld {
		replicaCount = 3 // Deployment should start with "bau" in the test. Therefore 3
	} else {
		replicaCount = 1
	}

	var appName = "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": strconv.FormatBool(optInOld),
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     "3",
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	matchlabels := map[string]string{
		"app": appName,
	}
	var deploymentname string
	deploymentname = "case" + strconv.Itoa(casenumber)

	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentname,
			Namespace:   deploymentInfo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},

		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Replicas: &replicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "chriscmsoft/random-generator:latest",
						Name:  deploymentInfo.Name},
					},
				},
			},
		},
	}
	return *dep
}
