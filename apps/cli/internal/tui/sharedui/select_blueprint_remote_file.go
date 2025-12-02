package sharedui

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/consts"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
	clistrings "github.com/newstack-cloud/bluelink/apps/cli/strings"
)

type SelectBlueprintRemoteFileModel struct {
	styles            *styles.BluelinkStyles
	selectedFile      string
	source            *string
	urlForm           *huh.Form
	objectStorageForm *huh.Form
}

func (m SelectBlueprintRemoteFileModel) Init() tea.Cmd {
	return tea.Batch(m.urlForm.Init(), m.objectStorageForm.Init())
}

func (m SelectBlueprintRemoteFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case SelectBlueprintSourceMsg:
		*m.source = msg.Source
	}

	if clistrings.FromPointer(m.source) == consts.BlueprintSourceHTTPS {
		urlFormModel, cmd := m.urlForm.Update(msg)
		if form, ok := urlFormModel.(*huh.Form); ok {
			m.urlForm = form
			cmds = append(cmds, cmd)
		}

		if m.urlForm.State == huh.StateCompleted {
			cmds = append(
				cmds,
				selectBlueprintCmd(
					m.urlForm.GetString("blueprintFileUrl"),
					consts.BlueprintSourceHTTPS,
				),
			)
		}
	} else {
		objectStorageFormModel, cmd := m.objectStorageForm.Update(msg)
		if form, ok := objectStorageFormModel.(*huh.Form); ok {
			m.objectStorageForm = form
			cmds = append(cmds, cmd)
		}

		if m.objectStorageForm.State == huh.StateCompleted {
			cmds = append(
				cmds,
				selectBlueprintCmd(
					createObjectPathWithBucketName(m.objectStorageForm),
					consts.BlueprintSourceS3,
				),
			)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SelectBlueprintRemoteFileModel) View() string {
	if clistrings.FromPointer(m.source) == consts.BlueprintSourceHTTPS {
		return m.urlForm.View()
	}

	return m.objectStorageForm.View()
}

func NewSelectBlueprintRemoteFile(
	blueprintFile string,
	bluelinkStyles *styles.BluelinkStyles,
) *SelectBlueprintRemoteFileModel {
	initSource := ""

	model := &SelectBlueprintRemoteFileModel{
		styles:       bluelinkStyles,
		selectedFile: blueprintFile,
		source:       &initSource,
	}

	model.urlForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("blueprintFileUrl").
				Title("HTTPS resource URL blueprint file location").
				Description("The public URL of the blueprint to download.").
				Placeholder("https://assets.example.com/project.blueprint.yml").
				Validate(func(value string) error {
					if !isValidHTTPSURL(value) {
						return errors.New("please provide a valid HTTPS URL")
					}

					return nil
				}).
				WithWidth(80),
		),
	).WithTheme(styles.NewBluelinkHuhTheme())

	model.objectStorageForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("bucketName").
				TitleFunc(func() string {
					return bucketNameTitle(clistrings.FromPointer(model.source))
				}, model.source).
				DescriptionFunc(func() string {
					return bucketNameDescription(clistrings.FromPointer(model.source))
				}, model.source).
				PlaceholderFunc(func() string {
					return bucketNamePlaceholder(clistrings.FromPointer(model.source))
				}, model.source).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("bucket name cannot be empty")
					}

					return nil
				}).
				WithWidth(80),
			huh.NewInput().
				Key("objectPath").
				Title("Object path").
				DescriptionFunc(func() string {
					return objectPathDescription(clistrings.FromPointer(model.source))
				}, model.source).
				Placeholder("path/to/example.blueprint.yml").
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("object path cannot be empty")
					}

					return nil
				}).
				WithWidth(80),
		),
	).WithTheme(styles.NewBluelinkHuhTheme())

	return model
}

func bucketNameTitle(source string) string {
	switch source {
	case consts.BlueprintSourceS3:
		return "S3 bucket"
	case consts.BlueprintSourceGCS:
		return "Google Cloud Storage bucket"
	case consts.BlueprintSourceAzureBlob:
		return "Azure Blob Storage container"
	}
	return ""
}

func bucketNameDescription(source string) string {
	switch source {
	case consts.BlueprintSourceS3:
		return "The name of the S3 bucket containing the blueprint file."
	case consts.BlueprintSourceGCS:
		return "The name of the GCS bucket containing the blueprint file."
	case consts.BlueprintSourceAzureBlob:
		return "The name of the Azure Blob Storage container containing the blueprint file."
	}
	return ""
}

func bucketNamePlaceholder(source string) string {
	switch source {
	case consts.BlueprintSourceS3:
		return "s3-bucket"
	case consts.BlueprintSourceGCS:
		return "gcs-bucket"
	case consts.BlueprintSourceAzureBlob:
		return "azure-blob-container"
	}
	return ""
}

func objectPathDescription(source string) string {
	switch source {
	case consts.BlueprintSourceS3:
		return "The path of the blueprint file in the S3 bucket."
	case consts.BlueprintSourceGCS:
		return "The path of the blueprint file in the GCS bucket."
	case consts.BlueprintSourceAzureBlob:
		return "The path of the blueprint file in the Azure Blob Storage container."
	}
	return ""
}

func isValidHTTPSURL(inputURL string) bool {
	path, err := url.Parse(inputURL)
	isValid := err == nil && path.Scheme == "https" && path.Host != ""
	return isValid
}

func createObjectPathWithBucketName(form *huh.Form) string {
	bucketName := form.GetString("bucketName")
	if !strings.HasSuffix(bucketName, "/") {
		bucketName = fmt.Sprintf("%s/", bucketName)
	}

	return fmt.Sprintf("%s%s", bucketName, form.GetString("objectPath"))
}
