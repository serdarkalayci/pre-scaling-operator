package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Deployment Watch Controller", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1

	var deployment v1.Deployment
	var clusterscalingstate *v1alpha1.ClusterScalingState
	BeforeEach(func() {
		key := types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		}

		// failed test runs that don't clean up leave resources behind.
		var replicas int32 = 2
		var replicaPoint *int32 = &replicas
		deployment = CreateDeployment(*replicaPoint, key, true)
		clusterscalingstate = CreateClusterScalingState()
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Deployment replica update", func() {
		It("Should Update Deployment replica count correctly", func() {
			By("Creating the deployment object first")
			Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), clusterscalingstate)).Should(Succeed())
			time.Sleep(time.Second * 5)

			key := types.NamespacedName{
				Name:      "test",
				Namespace: "default",
			}
			var equal int32 = 2
			fetched := &v1.Deployment{}
			Eventually(func() int32 {
				k8sClient.Get(context.Background(), key, fetched)
				return *fetched.Spec.Replicas
			}, timeout, interval).Should(Equal(equal))

			By("Deployment Created successfully")

		})
	})
})

func CreateClusterScalingState() *v1alpha1.ClusterScalingState {

	scalingState := &v1alpha1.ClusterScalingState{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterscalingstate-sample",
		},
		Spec: v1alpha1.ClusterScalingStateSpec{
			State: "bau",
		},
	}

	return scalingState
}

func CreateDeployment(replicaCount int32, deploymentInfo types.NamespacedName, optIn bool) v1.Deployment {

	appName := "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": strconv.FormatBool(optIn),
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     "4",
		"scaler/state-default-replicas": "1",
		"scaler/state-peak-replicas":    "5",
	}

	matchlabels := map[string]string{
		"app": appName,
	}
	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentInfo.Name,
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
