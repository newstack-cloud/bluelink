package languageservices

import (
	"os"
	"path"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
)

const blueprintURI = "file:///blueprint.yaml"

func loadTestBlueprintContent(blueprintFileName string) (string, error) {
	bytes, err := os.ReadFile(path.Join("__testdata", blueprintFileName))
	return string(bytes), err
}

// testBlueprintInfo holds blueprint data for completion tests.
type testBlueprintInfo struct {
	blueprint *schema.Blueprint
	tree      *schema.TreeNode
	content   string
}

// toDocumentContext converts testBlueprintInfo to a DocumentContext for testing.
func (info *testBlueprintInfo) toDocumentContext() *docmodel.DocumentContext {
	docCtx := docmodel.NewDocumentContextFromSchema(
		blueprintURI,
		info.blueprint,
		info.tree,
	)
	docCtx.Content = info.content
	return docCtx
}

func loadCompletionBlueprintAndTree(name string) (*testBlueprintInfo, error) {
	// Load and parse the blueprint content before the completion trigger character.
	// This is required as when a completion trigger character is entered, the current
	// state of the document will not be successfully parsed and the completion service
	// will be working with the parsed version of the document before the trigger character
	// was entered.
	contentBefore, err := loadTestBlueprintContent(path.Join(name, "before-completion-trigger.yaml"))
	if err != nil {
		return nil, err
	}

	blueprint, err := schema.LoadString(contentBefore, schema.YAMLSpecFormat)
	if err != nil {
		return nil, err
	}

	tree := schema.SchemaToTree(blueprint)

	// Load the content after the completion trigger character.
	afterTriggerContent, err := loadTestBlueprintContent(path.Join(name, "after-completion-trigger.yaml"))
	if err != nil {
		return nil, err
	}

	return &testBlueprintInfo{
		blueprint: blueprint,
		tree:      tree,
		content:   afterTriggerContent,
	}, nil
}

func completionItemLabels(completionItems []*lsp.CompletionItem) []string {
	labels := make([]string, len(completionItems))
	for i, item := range completionItems {
		labels[i] = item.Label
	}
	slices.Sort(labels)
	return labels
}

func sortCompletionItems(completionItems []*lsp.CompletionItem) []*lsp.CompletionItem {
	items := make([]*lsp.CompletionItem, len(completionItems))
	copy(items, completionItems)
	slices.SortFunc(items, func(a, b *lsp.CompletionItem) int {
		if a.Label < b.Label {
			return -1
		} else if a.Label > b.Label {
			return 1
		}
		return 0
	})
	return items
}

func strPtr(s string) *string {
	return &s
}
