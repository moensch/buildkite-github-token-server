package buildkite

type PipelineCreateInput struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Visibility     string `json:"visibility" default:"PRIVATE"`
	Description    string `json:"description"`
	Steps          struct {
		YAML string `json:"yaml"`
	} `json:"steps"`
	Repository struct {
		URL string `json:"url"`
	} `json:"repository"`
	Teams         []PipelineTeamAssignmentInput `json:"teams"`
	DefaultBranch string                        `json:"defaultBranch" default:"main"`
}

type PipelineAccessLevel string

const (
	PipelineAccessLevelManage       PipelineAccessLevel = "MANAGE_BUILD_AND_READ"
	PipelineAccessLevelBuildAndRead PipelineAccessLevel = "BUILD_AND_READ"
	PipelineAccessLevelReadOnly     PipelineAccessLevel = "READ_ONLY"
)

type PipelineTeamAssignmentInput struct {
	ID          string              `json:"id"`
	AccessLevel PipelineAccessLevel `json:"accessLevel"`
}

type PipelineCreateWebhookInput struct {
	ID string `json:"id"`
}
type BuildAnnotateInput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	// BuildID is the GraphQL ID of the build (not the UUID)
	BuildID string          `json:"buildID"`
	Body    string          `json:"body"`
	Style   AnnotationStyle `json:"style" default:"DEFAULT"`
	Context string          `json:"context,omitempty" default:"default"`
	Append  bool            `json:"append,omitempty"`
}

type AnnotationStyle string

const (
	AnnotationStyleDefault AnnotationStyle = "DEFAULT"
	AnnotationStyleSuccess AnnotationStyle = "SUCCESS"
	AnnotationStyleInfo    AnnotationStyle = "INFO"
	AnnotationStyleWarning AnnotationStyle = "WARNING"
	AnnotationStyleError   AnnotationStyle = "ERROR"
)

type BuildState string

const (
	BuildStateSkipped   BuildState = "SKIPPED"
	BuildStateCreating  BuildState = "CREATING"
	BuildStateScheduled BuildState = "SCHEDULED"
	BuildStateRunning   BuildState = "RUNNING"
	BuildStatePassed    BuildState = "PASSED"
	BuildStateFailed    BuildState = "FAILED"
	BuildStateCanceling BuildState = "CANCELING"
	BuildStateCanceled  BuildState = "CANCELED"
	BuildStateBlocked   BuildState = "BLOCKED"
	BuildStateNotRun    BuildState = "NOT_RUN"
)

type JobState string

const (
	JobStateBlocked   JobState = "BLOCKED"
	JobStateUnblocked JobState = "UNBLOCKED"
)
