/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Exclusive topology annotation is used to specify the topology which
	// be used for 1:1 exclusive scheduling.
	ExclusiveKeyAnnotationKey string = "leaderworkerset.sigs.k8s.io/exclusive-topology"

	// Set name label will record the leaderworkerset name that those resources
	// (Pod/Service/StatefulSets) belong to.
	SetNameLabelKey string = "leaderworkerset.sigs.k8s.io/name"

	// Group index label will be added to pods to record which group this
	// statefulset/pod belong to.
	GroupIndexLabelKey string = "leaderworkerset.sigs.k8s.io/group-index"

	// Worker index will be added to pods as a label and annotation which is
	// the index/identity of the pod in the group.
	WorkerIndexLabelKey string = "leaderworkerset.sigs.k8s.io/worker-index"

	// Size will be added to leader pods as an annotation which corresponds to
	// LeaderWorkerSet.Spec.LeaderWorkerTemplate.Size.
	SizeAnnotationKey string = "leaderworkerset.sigs.k8s.io/size"

	// Pods that are in the same group will have an annotation that is a unique
	// hash value.
	GroupUniqueHashLabelKey string = "leaderworkerset.sigs.k8s.io/group-key"

	// Worker pods will have an annotation that is the leader pod's name.
	LeaderPodNameAnnotationKey string = "leaderworkerset.sigs.k8s.io/leader-name"
)

// One group consists of a single leader and M workers, and the total number of pods in a group is M+1.
// LeaderWorkerSet will create N replicas of leader-worker pod groups (hereinafter referred to as group).
//
// Each group has a unique index between 0 and N-1. We call this the leaderIndex.
// The leaderIndex is used to uniquely name the leader pod of each group in the following format:
// leaderWorkerSetName-leaderIndex. This is considered as the name of the group too.
//
// Each worker pod in the group has a unique workerIndex between 1 and M. The leader also
// gets a workerIndex, and it is always set to 0.
// Worker pods are named using the format: leaderWorkerSetName-leaderIndex-workerIndex.
type LeaderWorkerSetSpec struct {
	// Number of leader-workers groups. A scale subresource is available to enable HPA. The
	// selector for HPA will be that of the leader pod, and so practically HPA will be looking up the
	// leader pod metrics. Note that the leader pod could aggregate metrics from
	// the rest of the group and expose them as a summary custom metric representing the whole
	// group.
	// On scale down, the leader pod as well as the workers statefulset will be deleted.
	Replicas *int32 `json:"replicas,omitempty"`

	// LeaderWorkerTemplate defines the template for leader/worker pods
	LeaderWorkerTemplate LeaderWorkerTemplate `json:"leaderWorkerTemplate"`
}

// Template of the leader/worker pods, the group will include at least one leader pod.
// Defaults to the worker template if not specified. The idea is to allow users to create a
// group with identical templates without needing to specify the template in both places.
// For the leader it represents the id of the group, while for the workers it represents the
// index within the group. For this reason, users should depend on the labels injected by this
// API whenever possible.
type LeaderWorkerTemplate struct {
	// LeaderTemplate defines the pod template for leader pods.
	LeaderTemplate *corev1.PodTemplateSpec `json:"leaderTemplate,omitempty"`

	// WorkerTemplate defines the pod template for worker pods.
	WorkerTemplate corev1.PodTemplateSpec `json:"workerTemplate"`

	// Number of pods to create. It is the total number of pods in each group.
	// The minimum is 1 which represent the leader. When set to 1, the leader
	// pod is created for each group as well as a 0-replica StatefulSet for the workers.
	Size *int32 `json:"size"`

	// RestartPolicy defines the restart policy when pod failures happen.
	RestartPolicy RestartPolicyType `json:"restartPolicy,omitempty"`
}

type RestartPolicyType string

const (
	// RecreateGroupOnPodRestart will recreate all the pods in the group if
	// 1. Any individual pod in the group is recreated; 2. Any containers/init-containers
	// in a pod is restarted. This is to ensure all pods/containers in the group will be
	// started in the same time.
	RecreateGroupOnPodRestart RestartPolicyType = "RecreateGroupOnPodRestart"

	// Default will follow the same behavior as the StatefulSet where only the failed pod
	// will be restarted on failure and other pods in the group will not be impacted.
	Default RestartPolicyType = "Default"
)

// LeaderWorkerSetStatus defines the observed state of LeaderWorkerSet
type LeaderWorkerSetStatus struct {
	// Conditions track the condition of the leaderworkerset.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ReadyReplicas track the number of groups that are in ready state.
	ReadyReplicas int `json:"readyReplicas,omitempty"`

	// Replicas track the active total number of groups.
	Replicas int `json:"replicas,omitempty"`

	// HPAPodSelector for pods that belong to the LeaderWorkerSet object, this is
	// needed for HPA to know what pods belong to the LeaderWorkerSet object. Here
	// we only select the leader pods.
	HPAPodSelector string `json:"hpaPodSelector,omitempty"`
}

type LeaderWorkerSetConditionType string

// These are built-in conditions of a LWS.
const (
	// LeaderWorkerSetAvailable means the lws is available, ie, at least the
	// minimum available groups are up and running.
	LeaderWorkerSetAvailable LeaderWorkerSetConditionType = "Available"

	// LeaderWorkerSetProgressing means lws is progressing. Progress for a
	// lws replica is considered when a new group is created, and when new pods
	// scale up and down. Before a group has all its pods ready, the group itself
	// will be in progressing state. And any group in progress will make
	// the lws as progressing state.
	LeaderWorkerSetProgressing LeaderWorkerSetConditionType = "Progressing"
)

// +genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LeaderWorkerSet is the Schema for the leaderworkersets API
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.hpaPodSelector
type LeaderWorkerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LeaderWorkerSetSpec   `json:"spec,omitempty"`
	Status LeaderWorkerSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LeaderWorkerSetList contains a list of LeaderWorkerSet.
type LeaderWorkerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LeaderWorkerSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LeaderWorkerSet{}, &LeaderWorkerSetList{})
}