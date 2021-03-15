package cmd

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	scribev1alpha1 "github.com/backube/scribe/api/v1alpha1"
)

type commonOptions struct {
	capacity            *resource.Quantity
	copyMethod          scribev1alpha1.CopyMethodType
	accessModes         []corev1.PersistentVolumeAccessMode
	serviceType         corev1.ServiceType
	externalSpec        *scribev1alpha1.ReplicationDestinationExternalSpec
	address             *string
	sshKeysSecret       *string
	sshUser             *string
	path                *string
	storageClassName    *string
	volumeSnapClassName *string
	pvc                 *string
	port                *int32
	parameters          map[string]string
}
