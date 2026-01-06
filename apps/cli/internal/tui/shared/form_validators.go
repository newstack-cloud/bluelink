package shared

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

// ValidateInstanceName validates an instance name.
// Returns an error if the name is empty, too short (< 3), or too long (> 128).
func ValidateInstanceName(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("instance name cannot be empty")
	}
	if len(trimmed) < 3 {
		return errors.New("instance name must be at least 3 characters")
	}
	if len(trimmed) > 128 {
		return errors.New("instance name must be at most 128 characters")
	}
	return nil
}

// ValidateChangesetID validates a changeset ID.
// Returns an error if the ID is empty.
func ValidateChangesetID(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("changeset ID is required when not staging first")
	}
	return nil
}

// NewInstanceNameInput creates a standard instance name input field.
func NewInstanceNameInput(valuePtr *string, description string) *huh.Input {
	return huh.NewInput().
		Key("instanceName").
		Title("Instance Name").
		Description(description).
		Placeholder("my-app-production").
		Value(valuePtr).
		Validate(ValidateInstanceName)
}

// NewInstanceIDNote creates a read-only note displaying an instance ID.
func NewInstanceIDNote(instanceID string) *huh.Note {
	return huh.NewNote().
		Title("Instance ID").
		Description(instanceID)
}

// NewChangesetIDGroup creates a changeset ID input group that hides when stageFirst is true.
func NewChangesetIDGroup(valuePtr *string, stageFirstPtr *bool, description string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Key("changesetID").
			Title("Changeset ID").
			Description(description).
			Placeholder("changeset-abc123").
			Value(valuePtr).
			Validate(ValidateChangesetID),
	).WithHideFunc(func() bool {
		return *stageFirstPtr
	})
}

// NewAutoApproveGroup creates an auto-approve confirm group that shows when stageFirst is true.
func NewAutoApproveGroup(valuePtr *bool, stageFirstPtr *bool, negativeLabel string) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Key("autoApprove").
			Title("Auto-approve staged changes?").
			Description("Skip confirmation after staging.").
			Affirmative("Yes, skip confirmation").
			Negative(negativeLabel).
			WithButtonAlignment(lipgloss.Left).
			Value(valuePtr),
	).WithHideFunc(func() bool {
		return !*stageFirstPtr
	})
}

// NewStageFirstConfirm creates a stage-first confirm field.
func NewStageFirstConfirm(valuePtr *bool, title, description string) *huh.Confirm {
	return huh.NewConfirm().
		Key("stageFirst").
		Title(title).
		Description(description).
		Affirmative("Yes, stage now").
		Negative("No, use existing changeset").
		WithButtonAlignment(lipgloss.Left).
		Value(valuePtr)
}

// NewThemedForm creates a new form with the standard theme applied.
func NewThemedForm(styles *stylespkg.Styles, groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(stylespkg.NewHuhTheme(styles.Palette))
}
