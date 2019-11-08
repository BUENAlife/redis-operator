package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisfailoverv1 "github.com/spotahome/redis-operator/api/redisfailover/v1"
	"github.com/spotahome/redis-operator/log"
	mK8SService "github.com/spotahome/redis-operator/mocks/service/k8s"
	rfservice "github.com/spotahome/redis-operator/operator/redisfailover/service"
)

func TestRedisStatefulSetStorageGeneration(t *testing.T) {
	configMapName := rfservice.GetRedisName(generateRF())
	shutdownConfigMapName := rfservice.GetRedisShutdownConfigMapName(generateRF())
	executeMode := int32(0744)
	tests := []struct {
		name           string
		ownerRefs      []metav1.OwnerReference
		expectedSS     appsv1.StatefulSet
		rfRedisStorage redisfailoverv1.RedisStorage
	}{
		{
			name: "Default values",
			expectedSS: appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "redis-config",
											MountPath: "/redis",
										},
										{
											Name:      "redis-shutdown-config",
											MountPath: "/redis-shutdown",
										},
										{
											Name:      "redis-data",
											MountPath: "/data",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "redis-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMapName,
											},
										},
									},
								},
								{
									Name: "redis-shutdown-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: shutdownConfigMapName,
											},
											DefaultMode: &executeMode,
										},
									},
								},
								{
									Name: "redis-data",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
							},
						},
					},
				},
			},
			rfRedisStorage: redisfailoverv1.RedisStorage{},
		},
		{
			name: "Defined an emptydir with storage on memory",
			expectedSS: appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "redis-config",
											MountPath: "/redis",
										},
										{
											Name:      "redis-shutdown-config",
											MountPath: "/redis-shutdown",
										},
										{
											Name:      "redis-data",
											MountPath: "/data",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "redis-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMapName,
											},
										},
									},
								},
								{
									Name: "redis-shutdown-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: shutdownConfigMapName,
											},
											DefaultMode: &executeMode,
										},
									},
								},
								{
									Name: "redis-data",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{
											Medium: corev1.StorageMediumMemory,
										},
									},
								},
							},
						},
					},
				},
			},
			rfRedisStorage: redisfailoverv1.RedisStorage{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			name: "Defined an persistentvolumeclaim",
			expectedSS: appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "redis-config",
											MountPath: "/redis",
										},
										{
											Name:      "redis-shutdown-config",
											MountPath: "/redis-shutdown",
										},
										{
											Name:      "pvc-data",
											MountPath: "/data",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "redis-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMapName,
											},
										},
									},
								},
								{
									Name: "redis-shutdown-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: shutdownConfigMapName,
											},
											DefaultMode: &executeMode,
										},
									},
								},
							},
						},
					},
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-data",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									"ReadWriteOnce",
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					},
				},
			},
			rfRedisStorage: redisfailoverv1.RedisStorage{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							"ReadWriteOnce",
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
		{
			name: "Defined an persistentvolumeclaim with ownerRefs",
			ownerRefs: []metav1.OwnerReference{
				{
					Name: "testing",
				},
			},
			expectedSS: appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "redis-config",
											MountPath: "/redis",
										},
										{
											Name:      "redis-shutdown-config",
											MountPath: "/redis-shutdown",
										},
										{
											Name:      "pvc-data",
											MountPath: "/data",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "redis-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMapName,
											},
										},
									},
								},
								{
									Name: "redis-shutdown-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: shutdownConfigMapName,
											},
											DefaultMode: &executeMode,
										},
									},
								},
							},
						},
					},
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-data",
								OwnerReferences: []metav1.OwnerReference{
									{
										Name: "testing",
									},
								},
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									"ReadWriteOnce",
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					},
				},
			},
			rfRedisStorage: redisfailoverv1.RedisStorage{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							"ReadWriteOnce",
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
		{
			name: "Defined an persistentvolumeclaim with ownerRefs keeping the pvc",
			ownerRefs: []metav1.OwnerReference{
				{
					Name: "testing",
				},
			},
			expectedSS: appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "redis-config",
											MountPath: "/redis",
										},
										{
											Name:      "redis-shutdown-config",
											MountPath: "/redis-shutdown",
										},
										{
											Name:      "pvc-data",
											MountPath: "/data",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "redis-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMapName,
											},
										},
									},
								},
								{
									Name: "redis-shutdown-config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: shutdownConfigMapName,
											},
											DefaultMode: &executeMode,
										},
									},
								},
							},
						},
					},
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-data",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									"ReadWriteOnce",
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					},
				},
			},
			rfRedisStorage: redisfailoverv1.RedisStorage{
				KeepAfterDeletion: true,
				PersistentVolumeClaim: &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							"ReadWriteOnce",
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required storage
		rf := generateRF()
		rf.Spec.Redis.Storage = test.rfRedisStorage

		generatedStatefulSet := appsv1.StatefulSet{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateStatefulSet", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			ss := args.Get(1).(*appsv1.StatefulSet)
			generatedStatefulSet = *ss
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureRedisStatefulset(rf, nil, test.ownerRefs)

		// Check that the storage-related fields are as spected
		assert.Equal(test.expectedSS.Spec.Template.Spec.Volumes, generatedStatefulSet.Spec.Template.Spec.Volumes)
		assert.Equal(test.expectedSS.Spec.Template.Spec.Containers[0].VolumeMounts, generatedStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
		assert.Equal(test.expectedSS.Spec.VolumeClaimTemplates, generatedStatefulSet.Spec.VolumeClaimTemplates)
		assert.NoError(err)
	}
}

func TestRedisStatefulSetCommands(t *testing.T) {
	tests := []struct {
		name             string
		givenCommands    []string
		expectedCommands []string
	}{
		{
			name:          "Default values",
			givenCommands: []string{},
			expectedCommands: []string{
				"redis-server",
				"/redis/redis.conf",
			},
		},
		{
			name: "Given commands should be used in redis container",
			givenCommands: []string{
				"test",
				"command",
			},
			expectedCommands: []string{
				"test",
				"command",
			},
		},
	}

	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required storage
		rf := generateRF()
		rf.Spec.Redis.Command = test.givenCommands

		gotCommands := []string{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateStatefulSet", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			ss := args.Get(1).(*appsv1.StatefulSet)
			gotCommands = ss.Spec.Template.Spec.Containers[0].Command
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureRedisStatefulset(rf, nil, []metav1.OwnerReference{})

		assert.Equal(test.expectedCommands, gotCommands)
		assert.NoError(err)
	}
}

func TestSentinelDeploymentCommands(t *testing.T) {
	tests := []struct {
		name             string
		givenCommands    []string
		expectedCommands []string
	}{
		{
			name:          "Default values",
			givenCommands: []string{},
			expectedCommands: []string{
				"redis-server",
				"/redis/sentinel.conf",
				"--sentinel",
			},
		},
		{
			name: "Given commands should be used in sentinel container",
			givenCommands: []string{
				"test",
				"command",
			},
			expectedCommands: []string{
				"test",
				"command",
			},
		},
	}

	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required storage
		rf := generateRF()
		rf.Spec.Sentinel.Command = test.givenCommands

		gotCommands := []string{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateDeployment", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			d := args.Get(1).(*appsv1.Deployment)
			gotCommands = d.Spec.Template.Spec.Containers[0].Command
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureSentinelDeployment(rf, nil, []metav1.OwnerReference{})

		assert.Equal(test.expectedCommands, gotCommands)
		assert.NoError(err)
	}
}

func TestRedisStatefulSetPodAnnotations(t *testing.T) {
	tests := []struct {
		name                   string
		givenPodAnnotations    map[string]string
		expectedPodAnnotations map[string]string
	}{
		{
			name:                   "PodAnnotations was not defined",
			givenPodAnnotations:    nil,
			expectedPodAnnotations: nil,
		},
		{
			name: "PodAnnotations is defined",
			givenPodAnnotations: map[string]string{
				"some":               "annotation",
				"path/to/annotation": "here",
			},
			expectedPodAnnotations: map[string]string{
				"some":               "annotation",
				"path/to/annotation": "here",
			},
		},
	}

	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required annotations
		rf := generateRF()
		rf.Spec.Redis.PodAnnotations = test.givenPodAnnotations

		gotPodAnnotations := map[string]string{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateStatefulSet", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			ss := args.Get(1).(*appsv1.StatefulSet)
			gotPodAnnotations = ss.Spec.Template.ObjectMeta.Annotations
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureRedisStatefulset(rf, nil, []metav1.OwnerReference{})

		assert.Equal(test.expectedPodAnnotations, gotPodAnnotations)
		assert.NoError(err)
	}
}

func TestSentinelDeploymentPodAnnotations(t *testing.T) {
	tests := []struct {
		name                   string
		givenPodAnnotations    map[string]string
		expectedPodAnnotations map[string]string
	}{
		{
			name:                   "PodAnnotations was not defined",
			givenPodAnnotations:    nil,
			expectedPodAnnotations: nil,
		},
		{
			name: "PodAnnotations is defined",
			givenPodAnnotations: map[string]string{
				"some":               "annotation",
				"path/to/annotation": "here",
			},
			expectedPodAnnotations: map[string]string{
				"some":               "annotation",
				"path/to/annotation": "here",
			},
		},
	}

	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required annotations
		rf := generateRF()
		rf.Spec.Sentinel.PodAnnotations = test.givenPodAnnotations

		gotPodAnnotations := map[string]string{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateDeployment", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			d := args.Get(1).(*appsv1.Deployment)
			gotPodAnnotations = d.Spec.Template.ObjectMeta.Annotations
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureSentinelDeployment(rf, nil, []metav1.OwnerReference{})

		assert.Equal(test.expectedPodAnnotations, gotPodAnnotations)
		assert.NoError(err)
	}
}

func TestRedisStatefulSetLabels(t *testing.T) {
	tests := []struct {
		name           string
		givenLabels    map[string]string
		expectedLabels map[string]string
	}{
		{
			name: "No given labels",
			expectedLabels: map[string]string{
				"deployment-version":          "1",
				"app.kubernetes.io/name":      "test",
				"app.kubernetes.io/component": "redis",
				"app.kubernetes.io/part-of":   "redis-failover",
			},
		},
		{
			name:        "Some given label",
			givenLabels: map[string]string{"someotherlabel": "value"},
			expectedLabels: map[string]string{
				"someotherlabel":              "value",
				"deployment-version":          "1",
				"app.kubernetes.io/name":      "test",
				"app.kubernetes.io/component": "redis",
				"app.kubernetes.io/part-of":   "redis-failover",
			},
		},
		{
			name:        "No overwrite label",
			givenLabels: map[string]string{"deployment-version": "10"},
			expectedLabels: map[string]string{
				"deployment-version":          "10",
				"app.kubernetes.io/name":      "test",
				"app.kubernetes.io/component": "redis",
				"app.kubernetes.io/part-of":   "redis-failover",
			},
		},
	}
	for _, test := range tests {
		assert := assert.New(t)

		// Generate a default RedisFailover and attaching the required annotations
		rf := generateRF()

		gotPodLabels := map[string]string{}

		ms := &mK8SService.Services{}
		ms.On("CreateOrUpdatePodDisruptionBudget", namespace, mock.Anything).Once().Return(nil, nil)
		ms.On("CreateOrUpdateStatefulSet", namespace, mock.Anything).Once().Run(func(args mock.Arguments) {
			ss := args.Get(1).(*appsv1.StatefulSet)
			gotPodLabels = ss.Spec.Template.ObjectMeta.Labels
		}).Return(nil)

		client := rfservice.NewRedisFailoverKubeClient(ms, log.Dummy)
		err := client.EnsureRedisStatefulset(rf, test.givenLabels, []metav1.OwnerReference{})

		assert.Equal(test.expectedLabels, gotPodLabels)
		assert.NoError(err)

	}
}
