package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"regexp"
	"strconv"
)

func GetPodName(migratorName string, sourcePodName string, initialFlag bool) (string, error) {
	if !initialFlag {
		// return targetPodName:sourcePodNameNumber + 1
		re := regexp.MustCompile("[0-9]+")
		numberStr := re.FindAllString(sourcePodName, -1)
		number, err := strconv.ParseInt(numberStr[len(numberStr)-1], 10, 64)
		return migratorName + "-pod-" + strconv.FormatInt(number+1, 10), err
	} else {
		// return sourcePodName
		return migratorName + "-pod-0", nil
	}

}
func GetPodFromTemplate(template *corev1.PodTemplateSpec, namespace string) *corev1.Pod {
	desiredLabels := getPodsLabelSet(template)
	desiredFinalizers := getPodsFinalizers(template)
	desiredAnnotations := getPodsAnnotationSet(template)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Labels:      desiredLabels,
			Finalizers:  desiredFinalizers,
			Annotations: desiredAnnotations,
		},
	}
	pod.Spec = *template.Spec.DeepCopy()
	return pod
}

func getPodsLabelSet(template *corev1.PodTemplateSpec) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	return desiredLabels
}

func getPodsFinalizers(template *corev1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}

func getPodsAnnotationSet(template *corev1.PodTemplateSpec) labels.Set {
	desiredAnnotations := make(labels.Set)
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}
	return desiredAnnotations
}
