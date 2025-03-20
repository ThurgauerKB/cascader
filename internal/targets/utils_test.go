package targets

import (
	"testing"

	"github.com/thurgauerkb/cascader/internal/utils"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateRestartedAtAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("No existing annotations", func(t *testing.T) {
		t.Parallel()

		template := &corev1.PodTemplateSpec{}

		updateRestartedAtAnnotation(template)

		annotations := template.GetAnnotations()
		assert.NotNil(t, annotations, "annotations map should be initialized")
		val, exists := annotations[utils.RestartedAtKey]
		assert.True(t, exists, "restartedAt annotation should exist")
		assert.NotEmpty(t, val, "restartedAt annotation should not be empty")
	})

	t.Run("Overwrites existing restartedAt annotation", func(t *testing.T) {
		t.Parallel()

		oldValue := "2023-01-01T00:00:00Z"
		template := &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					utils.RestartedAtKey: oldValue,
				},
			},
		}

		updateRestartedAtAnnotation(template)

		annotations := template.GetAnnotations()
		assert.NotNil(t, annotations)
		val := annotations[utils.RestartedAtKey]
		assert.NotEqual(t, oldValue, val, "restartedAt should be overwritten with a new value")
		assert.NotEmpty(t, val)
	})

	t.Run("Preserves other annotations", func(t *testing.T) {
		t.Parallel()

		template := &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"customKey": "customValue",
				},
			},
		}

		updateRestartedAtAnnotation(template)

		annotations := template.GetAnnotations()
		assert.NotNil(t, annotations)
		assert.Equal(t, "customValue", annotations["customKey"], "should not overwrite other annotations")

		restartedVal, exists := annotations[utils.RestartedAtKey]
		assert.True(t, exists, "should set restartedAt annotation")
		assert.NotEmpty(t, restartedVal)
	})
}
