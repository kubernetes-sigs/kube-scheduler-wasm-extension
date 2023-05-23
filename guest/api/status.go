package api

// StatusCode is the Status code/type which is returned from plugins.
type StatusCode int

// These are predefined codes used in a Status.
const (
	// StatusCodeSuccess means that plugin ran correctly and found pod schedulable.
	StatusCodeSuccess StatusCode = iota
	// StatusCodeError is used for internal plugin errors, unexpected input, etc.
	StatusCodeError
	// StatusCodeUnschedulable is used when a plugin finds a pod unschedulable. The scheduler might attempt to
	// run other postFilter plugins like preemption to get this pod scheduled.
	// Use StatusCodeUnschedulableAndUnresolvable to make the scheduler skipping other postFilter plugins.
	// The accompanying status message should explain why the pod is unschedulable.
	StatusCodeUnschedulable
	// StatusCodeUnschedulableAndUnresolvable is used when a plugin finds a pod unschedulable and
	// other postFilter plugins like preemption would not change anything.
	// Plugins should return StatusCodeUnschedulable if it is possible that the pod can get scheduled
	// after running other postFilter plugins.
	// The accompanying status message should explain why the pod is unschedulable.
	StatusCodeUnschedulableAndUnresolvable
	// StatusCodeWait is used when a Permit plugin finds a pod scheduling should wait.
	StatusCodeWait
	// StatusCodeSkip is used in the following scenarios:
	// - when a Bind plugin chooses to skip binding.
	// - when a PreFilter plugin returns StatusCodeSkip so that coupled Filter plugin/PreFilterExtensions() will be skipped.
	// - when a PreScore plugin returns StatusCodeSkip so that coupled Score plugin will be skipped.
	StatusCodeSkip
)
