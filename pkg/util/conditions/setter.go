/*
Copyright 2020 The Kubernetes Authors.

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

package conditions

import (
	"fmt"
	"sort"
	"time"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Setter interface defines methods that a Machine API object should implement in order to
// use the conditions package for setting conditions.
type Setter interface {
	Getter
	SetConditions(machinev1.Conditions)
}

// Set sets the given condition.
//
// NOTE: If a condition already exists, the LastTransitionTime is updated only if a change is detected
// in any of the following fields: Status, Reason, Severity and Message.
func Set(to interface{}, condition *machinev1.Condition) {
	if to == nil || condition == nil {
		return
	}

	obj := getSetterObject(to)

	// Check if the new conditions already exists, and change it only if there is a status
	// transition (otherwise we should preserve the current last transition time)-
	conditions := obj.GetConditions()
	exists := false
	for i := range conditions {
		existingCondition := conditions[i]
		if existingCondition.Type == condition.Type {
			exists = true
			if !hasSameState(&existingCondition, condition) {
				condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
				conditions[i] = *condition
				break
			}
			condition.LastTransitionTime = existingCondition.LastTransitionTime
			break
		}
	}

	// If the condition does not exist, add it, setting the transition time only if not already set
	if !exists {
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
		}
		conditions = append(conditions, *condition)
	}

	// Sorts conditions for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool {
		return lexicographicLess(&conditions[i], &conditions[j])
	})

	obj.SetConditions(conditions)
}

// TrueCondition returns a condition with Status=True and the given type.
func TrueCondition(t machinev1.ConditionType) *machinev1.Condition {
	return &machinev1.Condition{
		Type:   t,
		Status: corev1.ConditionTrue,
	}
}

// FalseCondition returns a condition with Status=False and the given type.
func FalseCondition(t machinev1.ConditionType, reason string, severity machinev1.ConditionSeverity, messageFormat string, messageArgs ...interface{}) *machinev1.Condition {
	return &machinev1.Condition{
		Type:     t,
		Status:   corev1.ConditionFalse,
		Reason:   reason,
		Severity: severity,
		Message:  fmt.Sprintf(messageFormat, messageArgs...),
	}
}

// UnknownCondition returns a condition with Status=Unknown and the given type.
func UnknownCondition(t machinev1.ConditionType, reason string, messageFormat string, messageArgs ...interface{}) *machinev1.Condition {
	return &machinev1.Condition{
		Type:    t,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: fmt.Sprintf(messageFormat, messageArgs...),
	}
}

// MarkTrue sets Status=True for the condition with the given type.
func MarkTrue(to interface{}, t machinev1.ConditionType) {
	obj := getSetterObject(to)
	Set(obj, TrueCondition(t))
}

// MarkFalse sets Status=False for the condition with the given type.
func MarkFalse(to interface{}, t machinev1.ConditionType, reason string, severity machinev1.ConditionSeverity, messageFormat string, messageArgs ...interface{}) {
	obj := getSetterObject(to)
	Set(obj, FalseCondition(t, reason, severity, messageFormat, messageArgs...))
}

func getSetterObject(from interface{}) Setter {
	switch obj := from.(type) {
	case machinev1.Machine:
		return &MachineWrapper{&obj}
	case machinev1.MachineHealthCheck:
		return &MachineHealthCheckWrapper{&obj}
	default:
		panic("type is not supported as conditions getter")
	}
}

// lexicographicLess returns true if a condition is less than another with regards to the
// to order of conditions designed for convenience of the consumer, i.e. kubectl.
func lexicographicLess(i, j *machinev1.Condition) bool {
	return i.Type < j.Type
}

// hasSameState returns true if a condition has the same state of another; state is defined
// by the union of following fields: Type, Status, Reason, Severity and Message (it excludes LastTransitionTime).
func hasSameState(i, j *machinev1.Condition) bool {
	return i.Type == j.Type &&
		i.Status == j.Status &&
		i.Reason == j.Reason &&
		i.Severity == j.Severity &&
		i.Message == j.Message
}
