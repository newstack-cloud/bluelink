package docmodel

import "github.com/newstack-cloud/bluelink/libs/blueprint/source"

// DuplicateKeyError represents a duplicate key found in a mapping.
type DuplicateKeyError struct {
	Key      string
	Range    source.Range
	KeyRange *source.Range
	IsFirst  bool
}

// DuplicateKeyResult holds all duplicate key errors found in a document.
type DuplicateKeyResult struct {
	Errors []*DuplicateKeyError
}

// DetectDuplicateKeys analyzes a UnifiedNode tree and returns all duplicate key errors.
func DetectDuplicateKeys(root *UnifiedNode) *DuplicateKeyResult {
	result := &DuplicateKeyResult{
		Errors: []*DuplicateKeyError{},
	}
	if root == nil {
		return result
	}
	detectDuplicatesRecursive(root, result)
	return result
}

func detectDuplicatesRecursive(node *UnifiedNode, result *DuplicateKeyResult) {
	if node == nil {
		return
	}

	if node.Kind == NodeKindMapping {
		checkMappingForDuplicates(node, result)
	}

	for _, child := range node.Children {
		detectDuplicatesRecursive(child, result)
	}
}

func checkMappingForDuplicates(mapping *UnifiedNode, result *DuplicateKeyResult) {
	keyOccurrences := make(map[string][]*UnifiedNode)

	for _, child := range mapping.Children {
		if child.FieldName != "" {
			keyOccurrences[child.FieldName] = append(keyOccurrences[child.FieldName], child)
		}
	}

	for key, occurrences := range keyOccurrences {
		if len(occurrences) <= 1 {
			continue
		}
		for i, node := range occurrences {
			result.Errors = append(result.Errors, &DuplicateKeyError{
				Key:      key,
				Range:    node.Range,
				KeyRange: node.KeyRange,
				IsFirst:  i == 0,
			})
		}
	}
}
