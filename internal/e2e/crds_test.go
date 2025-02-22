package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/validations"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("e2e Test for the crd controllers", func() {

	const (
		timeout  = time.Second * 100
		interval = time.Millisecond * 500
	)

	var (
		casenumber           = 1
		deploymentconfigList []ocv1.DeploymentConfig
		deploymentList       []v1.Deployment
		css                  v1alpha1.ClusterScalingState
		css1                 v1alpha1.ClusterScalingState
		ss                   v1alpha1.ScalingState
		cssd                 v1alpha1.ClusterScalingStateDefinition
		namespaceList        []corev1.Namespace

		key = types.NamespacedName{
			Name:      "test",
			Namespace: "e2e-tests-crds" + strconv.Itoa(casenumber),
		}
	)

	OpenshiftCluster, _ := validations.OpenshiftClusterCheck()

	BeforeEach(func() {

		cssd = CreateClusterScalingStateDefinition()

		Expect(k8sClient.Create(context.Background(), &cssd)).Should(Succeed())

		namespaceList = createMultipleNamespaces(key.Namespace, 2)

		for _, ns := range namespaceList {
			Expect(k8sClient.Create(context.Background(), &ns)).Should(Succeed())
		}

		if OpenshiftCluster {

			for _, ns := range namespaceList {

				deploymentconfigList = createMultipleDeploymentConfigs(ns.Name, 2, casenumber)

				for _, deploymentconfig := range deploymentconfigList {
					Expect(k8sClient.Create(context.Background(), &deploymentconfig)).Should(Succeed())
				}
			}

		} else {

			for _, ns := range namespaceList {

				deploymentList = createMultipleDeployments(ns.Name, 2, casenumber)

				for _, deployment := range deploymentList {
					Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())
				}
			}

		}

	})

	AfterEach(func() {
		// Wait until all potential wait-loops in the step scaler are finished.
		time.Sleep(time.Second * 5)

		if casenumber == 1 {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
		} else if casenumber == 2 {
			Expect(k8sClient.Delete(context.Background(), &ss)).Should(Succeed())
		} else if casenumber == 5 {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
		} else if casenumber == 6 {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), &css1)).Should(Succeed())
		} else {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), &ss)).Should(Succeed())
		}

		for _, ns := range namespaceList {
			Expect(k8sClient.Delete(context.Background(), &ns)).Should(Succeed())
		}
		time.Sleep(time.Second * 5)
		Expect(k8sClient.Delete(context.Background(), &cssd)).Should(Succeed())

		casenumber = casenumber + 1

		key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber)

		time.Sleep(time.Second * 2)
	})

	Context("Deployment in place and modification test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(expectedReplicas []int) {

				fetchedDeploymentConfigList := []ocv1.DeploymentConfig{}
				fetchedDeploymentList := []v1.Deployment{}

				if casenumber == 1 {
					css = CreateClusterScalingState(casenumber, "bau", "")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())
				} else if casenumber == 2 {
					ss = CreateScalingState("peak", namespaceList[0].Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				} else if casenumber == 3 {
					css = CreateClusterScalingState(casenumber, "bau", "")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 5)
					ss = CreateScalingState("peak", namespaceList[0].Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				} else if casenumber == 4 {
					css = CreateClusterScalingState(casenumber, "peak", "")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 5)

					ss = CreateScalingState("bau", namespaceList[1].Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				} else if casenumber == 5 {
					css = CreateClusterScalingState(casenumber, "peak", "prod-test")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 5)

				} else if casenumber == 6 {

					css = CreateClusterScalingState(casenumber, "bau", "prod-test")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					css1 = CreateClusterScalingState(casenumber, "peak", "prod-prod")
					Expect(k8sClient.Create(context.Background(), &css1)).Should(Succeed())

					time.Sleep(time.Second * 5)

				} else if casenumber == 7 {
					css = CreateClusterScalingState(casenumber, "bau", "")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					ss = CreateScalingState("peak", namespaceList[0].Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())

					time.Sleep(time.Second * 5)
					// get the cssd back to modify
					cssdList := &v1alpha1.ClusterScalingStateDefinitionList{}
					Eventually(func() v1alpha1.ClusterScalingStateDefinitionList {
						k8sClient.List(context.Background(), cssdList)
						return *cssdList
					}, timeout, interval).Should(Not(BeNil()))

					cssdMofified := getModifiedClusterScalingStateDefinition(cssdList.Items[0], false, true)
					Expect(k8sClient.Update(context.Background(), &cssdMofified)).Should(Succeed())
				} else if casenumber == 8 {
					css = CreateClusterScalingState(casenumber, "peak", "")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 5)

					// get the cssd back to modify
					cssdList := &v1alpha1.ClusterScalingStateDefinitionList{}
					Eventually(func() v1alpha1.ClusterScalingStateDefinitionList {
						k8sClient.List(context.Background(), cssdList)
						return *cssdList
					}, timeout, interval).Should(Not(BeNil()))

					cssdMofified := getModifiedClusterScalingStateDefinition(cssdList.Items[0], true, false)
					Expect(k8sClient.Update(context.Background(), &cssdMofified)).Should(Succeed())
				}

				// Give the operator time to get to the states
				time.Sleep(time.Second * 5)

				if OpenshiftCluster {
					for _, ns := range namespaceList {
						for _, dc := range deploymentconfigList {
							Eventually(func() ocv1.DeploymentConfig {
								k8sClient.Get(context.Background(), updateKey(ns.Name, dc.Name, key), &dc)
								return dc
							}, timeout, interval).Should(Not(BeNil()))

							fetchedDeploymentConfigList = append(fetchedDeploymentConfigList, dc)
						}
					}

					for k := 0; k < len(fetchedDeploymentConfigList); k++ {
						Eventually(func() int32 {
							k8sClient.Get(context.Background(), updateKey(fetchedDeploymentConfigList[k].Namespace, fetchedDeploymentConfigList[k].Name, key), &fetchedDeploymentConfigList[k])
							return fetchedDeploymentConfigList[k].Spec.Replicas
						}, timeout, interval).Should(Equal(int32(expectedReplicas[k])))
					}

				} else {
					for _, ns := range namespaceList {
						for _, dep := range deploymentList {
							Eventually(func() v1.Deployment {
								k8sClient.Get(context.Background(), updateKey(ns.Name, dep.Name, key), &dep)
								return dep
							}, timeout, interval).Should(Not(BeNil()))

							fetchedDeploymentList = append(fetchedDeploymentList, dep)
						}

					}

					for k := 0; k < len(fetchedDeploymentList); k++ {
						Eventually(func() int32 {
							k8sClient.Get(context.Background(), updateKey(fetchedDeploymentList[k].Namespace, fetchedDeploymentList[k].Name, key), &fetchedDeploymentList[k])
							return *fetchedDeploymentList[k].Spec.Replicas
						}, timeout, interval).Should(Equal(int32(expectedReplicas[k])))
					}
				}

			},
				// Structure:  ("Description of the case" , expectedReplicas)
				table.Entry("CASE 1  | Apply a CSS and affect only opted-in applications", []int{2, 1, 2, 1}),
				table.Entry("CASE 2  | Apply a SS on one namespace", []int{4, 1, 1, 1}),
				table.Entry("CASE 3  | Apply SS with higher prio than an existing CSS", []int{4, 1, 2, 1}),
				table.Entry("CASE 4  | Apply CSS with higher prio than an existing SS", []int{4, 1, 4, 1}),
				table.Entry("CASE 5  | Use only one clusterscalingclass for both opted deployments", []int{4, 1, 4, 1}),
				table.Entry("CASE 6  | Use one clusterscalingclass each for both opted in deployments", []int{4, 1, 2, 1}),
				//table.Entry("CASE 7  | Swap Prio in CSSD", []int{2, 1, 2, 1}),
				//table.Entry("CASE 8  | Remove states in CSSD", []int{4, 1, 4, 1}),
			)
		})
	})

})

func createMultipleDeployments(namespaceName string, numberOfDCs, casenumber int) []v1.Deployment {

	var deps []v1.Deployment
	var optin bool

	for i := 1; i <= numberOfDCs; i++ {
		optin = !optin
		deployment := defineDeployment(namespaceName, strconv.FormatBool(optin), casenumber, i)
		deps = append(deps, deployment)

	}
	return deps
}

func updateKey(namespaceName, name string, key types.NamespacedName) types.NamespacedName {

	key.Name = name
	key.Namespace = namespaceName

	return key

}

func defineLabelsBasedOnCase(namespaceName string, optin string, casenumber int, number int) map[string]string {
	labels := map[string]string{}
	if casenumber == 5 {
		labels = map[string]string{
			"app":                  "random-generator-1",
			"scaler/scaling-class": "prod-test",
			"scaler/opt-in":        optin,
		}
	} else if namespaceName == "e2e-tests-crds6-1" && number == 1 {
		labels = map[string]string{
			"app":                  "random-generator-1",
			"scaler/scaling-class": "prod-prod",
			"scaler/opt-in":        optin,
		}
	} else if namespaceName == "e2e-tests-crds6-2" && number == 1 {
		labels = map[string]string{
			"app":                  "random-generator-1",
			"scaler/scaling-class": "prod-test",
			"scaler/opt-in":        optin,
		}
	} else if casenumber == 8 && number == 2 {
		labels = map[string]string{
			"app":                  "random-generator-1",
			"scaler/scaling-class": "prod-prod",
			"scaler/opt-in":        optin,
		}
	} else {
		labels = map[string]string{
			"app":           "random-generator-1",
			"scaler/opt-in": optin,
		}
	}

	return labels
}

func defineDeployment(namespaceName string, optin string, casenumber int, number int) v1.Deployment {

	var replicaCount int32 = 1

	labels := defineLabelsBasedOnCase(namespaceName, optin, casenumber, number)

	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber) + "-" + strconv.Itoa(number),
			Namespace: namespaceName,
			Labels:    labels,
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     "2",
				"scaler/state-default-replicas": "1",
				"scaler/state-peak-replicas":    "4",
			},
		},

		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "random-generator-1",
				},
			},
			Replicas: &replicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "random-generator-1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  "test",
						},
					},
				},
			},
		},
	}
	return *dep
}

func createMultipleDeploymentConfigs(namespaceName string, numberOfDCs, casenumber int) []ocv1.DeploymentConfig {

	var dcs []ocv1.DeploymentConfig
	var optin bool

	for i := 1; i <= numberOfDCs; i++ {
		optin = !optin
		deploymentconfig := defineDeploymentConfig(namespaceName, strconv.FormatBool(optin), casenumber, i)
		dcs = append(dcs, deploymentconfig)

	}
	return dcs
}

func defineDeploymentConfig(namespaceName string, optin string, casenumber int, number int) ocv1.DeploymentConfig {

	labels := defineLabelsBasedOnCase(namespaceName, optin, casenumber, number)

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber) + "-" + strconv.Itoa(number),
			Namespace: namespaceName,
			Labels:    labels,
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     "2",
				"scaler/state-default-replicas": "1",
				"scaler/state-peak-replicas":    "4",
			},
		},
		Spec: ocv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"deployment-config.name": "random-generator-1",
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment-config.name": "random-generator-1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  "test",
						},
					},
				},
			},
		},
	}
	return *deploymentConfig
}

func defineNS(namespaceName string, number int) corev1.Namespace {

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName + "-" + strconv.Itoa(number),
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	return *ns
}

func createMultipleNamespaces(namespaceName string, numberOfNamespaces int) []corev1.Namespace {

	var ns []corev1.Namespace

	for i := 1; i <= numberOfNamespaces; i++ {
		{
			namespace := defineNS(namespaceName, i)
			ns = append(ns, namespace)
		}
	}
	return ns
}
