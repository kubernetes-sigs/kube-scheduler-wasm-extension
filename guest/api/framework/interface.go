package framework

import (
	"context"
	"errors"
	"strings"

	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
)

// Plugin is the parent type for all the scheduling framework plugins.
type Plugin interface {
	Name() string
}

// FilterPlugin is an interface for Filter plugins. These plugins are called at the
// filter extension point for filtering out hosts that cannot run a pod.
// This concept used to be called 'predicate' in the original scheduler.
// These plugins should return "Success", "Unschedulable" or "Error" in Status.code.
// However, the scheduler accepts other valid codes as well.
// Anything other than "Success" will lead to exclusion of the given host from
// running the pod.
type FilterPlugin interface {
	Plugin
	// Filter is called by the scheduling framework.
	// All FilterPlugins should return "Success" to declare that
	// the given node fits the pod. If Filter doesn't return "Success",
	// it will return "Unschedulable", "UnschedulableAndUnresolvable" or "Error".
	// For the node being evaluated, Filter plugins should look at the passed
	// nodeInfo reference for this particular node's information (e.g., pods
	// considered to be running on the node) instead of looking it up in the
	// NodeInfoSnapshot because we don't guarantee that they will be the same.
	// For example, during preemption, we may pass a copy of the original
	// nodeInfo object that has some pods removed from it to evaluate the
	// possibility of preempting them to schedule the target pod.
	Filter(ctx context.Context, state *CycleState, pod PodFn, nodeInfo NodeInfoFn) *Status
}

type NodeInfoFn func() NodeInfo
type PodFn func() *protoapi.Pod

type Status struct {
	code    Code
	reasons []string
	err     error
}

// NewStatus makes a Status out of the given arguments and returns its pointer.
func NewStatus(code Code, reasons ...string) *Status {
	s := &Status{
		code:    code,
		reasons: reasons,
	}
	return s
}

// AsStatus wraps an error in a Status.
func AsStatus(err error) *Status {
	if err == nil {
		return nil
	}
	return &Status{
		code: Error,
		err:  err,
	}
}

func (s *Status) WithError(err error) *Status {
	s.err = err
	return s
}

// Code returns code of the Status.
func (s *Status) Code() Code {
	if s == nil {
		return Success
	}
	return s.code
}

// Message returns a concatenated message on reasons of the Status.
func (s *Status) Message() string {
	if s == nil {
		return ""
	}
	return strings.Join(s.Reasons(), ", ")
}

// Reasons returns reasons of the Status.
func (s *Status) Reasons() []string {
	if s.err != nil {
		return append([]string{s.err.Error()}, s.reasons...)
	}
	return s.reasons
}

// AppendReason appends given reason to the Status.
func (s *Status) AppendReason(reason string) {
	s.reasons = append(s.reasons, reason)
}

// IsSuccess returns true if and only if "Status" is nil or Code is "Success".
func (s *Status) IsSuccess() bool {
	return s.Code() == Success
}

// IsWait returns true if and only if "Status" is non-nil and its Code is "Wait".
func (s *Status) IsWait() bool {
	return s.Code() == Wait
}

// IsSkip returns true if and only if "Status" is non-nil and its Code is "Skip".
func (s *Status) IsSkip() bool {
	return s.Code() == Skip
}

// IsUnschedulable returns true if "Status" is Unschedulable (Unschedulable or UnschedulableAndUnresolvable).
func (s *Status) IsUnschedulable() bool {
	code := s.Code()
	return code == Unschedulable || code == UnschedulableAndUnresolvable
}

// AsError returns nil if the status is a success, a wait or a skip; otherwise returns an "error" object
// with a concatenated message on reasons of the Status.
func (s *Status) AsError() error {
	if s.IsSuccess() || s.IsWait() || s.IsSkip() {
		return nil
	}
	if s.err != nil {
		return s.err
	}
	return errors.New(s.Message())
}

// This list should be exactly the same as the codes iota defined above in the same order.
var codes = []string{"Success", "Error", "Unschedulable", "UnschedulableAndUnresolvable", "Wait", "Skip"}

func (c Code) String() string {
	return codes[c]
}

type CycleState struct {
	// TODO: https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/issues/16
}

type NodeInfo interface {
	Node() *protoapi.Node
}

// Code is the Status code/type which is returned from plugins.
type Code int

// These are predefined codes used in a Status.
const (
	// Success means that plugin ran correctly and found pod schedulable.
	// NOTE: A nil status is also considered as "Success".
	Success Code = iota
	// Error is used for internal plugin errors, unexpected input, etc.
	Error
	// Unschedulable is used when a plugin finds a pod unschedulable. The scheduler might attempt to
	// run other postFilter plugins like preemption to get this pod scheduled.
	// Use UnschedulableAndUnresolvable to make the scheduler skipping other postFilter plugins.
	// The accompanying status message should explain why the pod is unschedulable.
	Unschedulable
	// UnschedulableAndUnresolvable is used when a plugin finds a pod unschedulable and
	// other postFilter plugins like preemption would not change anything.
	// Plugins should return Unschedulable if it is possible that the pod can get scheduled
	// after running other postFilter plugins.
	// The accompanying status message should explain why the pod is unschedulable.
	UnschedulableAndUnresolvable
	// Wait is used when a Permit plugin finds a pod scheduling should wait.
	Wait
	// Skip is used in the following scenarios:
	// - when a Bind plugin chooses to skip binding.
	// - when a PreFilter plugin returns Skip so that coupled Filter plugin/PreFilterExtensions() will be skipped.
	// - when a PreScore plugin returns Skip so that coupled Score plugin will be skipped.
	Skip
)
