package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

func containsLSPPoint(bpRange *source.Range, lspPos lsp.Position, columnLeeway int) bool {
	bpPos := &source.Position{
		Line:   int(lspPos.Line + 1),
		Column: int(lspPos.Character + 1),
	}

	if bpRange.End == nil {
		return bpPos.Line > bpRange.Start.Line ||
			(bpPos.Line == bpRange.Start.Line && bpPos.Column >= bpRange.Start.Column-columnLeeway)
	}

	// Check in range on single line.
	if bpPos.Line == bpRange.Start.Line && bpPos.Line == bpRange.End.Line {
		return bpPos.Column >= bpRange.Start.Column-columnLeeway &&
			bpPos.Column <= bpRange.End.Column+columnLeeway
	}

	// Check in range on start line.
	if bpPos.Line == bpRange.Start.Line {
		return bpPos.Column >= bpRange.Start.Column-columnLeeway
	}

	// Check in range on end line.
	if bpPos.Line == bpRange.End.Line {
		return bpPos.Column <= bpRange.End.Column+columnLeeway
	}

	// Check in range across multiple lines.
	return bpPos.Line > bpRange.Start.Line && bpPos.Line < bpRange.End.Line
}

func lineContainsLSPPoint(bpRange *source.Range, lspPos lsp.Position) bool {
	bpPos := &source.Position{
		Line: int(lspPos.Line + 1),
	}

	if bpRange.End == nil {
		// Check >= to allow for root node that does not have an end position.
		return bpPos.Line >= bpRange.Start.Line
	}

	return bpPos.Line >= bpRange.Start.Line && bpPos.Line <= bpRange.End.Line
}

func rangeToLSPRange(bpRange *source.Range) *lsp.Range {
	if bpRange == nil {
		return nil
	}

	return &lsp.Range{
		Start: lsp.Position{
			Line:      uint32(bpRange.Start.Line - 1),
			Character: uint32(bpRange.Start.Column - 1),
		},
		End: lsp.Position{
			Line:      uint32(bpRange.End.Line - 1),
			Character: uint32(bpRange.End.Column - 1),
		},
	}
}

func collectElementsAtPosition(
	tree *schema.TreeNode,
	pos lsp.Position,
	logger *zap.Logger,
	collected *[]*schema.TreeNode,
	columnLeeway int,
) {
	if tree == nil {
		return
	}

	logger.Debug(
		"collectElementsAtPosition: checking element",
		zap.String("path", tree.Path),
		zap.Any("range", tree.Range),
		zap.Any("pos", pos),
	)
	if containsLSPPoint(tree.Range, pos, columnLeeway) {
		logger.Debug(
			"collectElementsAtPosition: found element",
			zap.String("path", tree.Path),
			zap.Any("range", tree.Range),
		)
		logger.Debug("Children length", zap.Int("length", len(tree.Children)))
		*collected = append(*collected, tree)
		i := 0
		for i < len(tree.Children) {
			collectElementsAtPosition(tree.Children[i], pos, logger, collected, columnLeeway)
			i += 1
		}
	}
}

func collectElementsOnLine(
	tree *schema.TreeNode,
	pos lsp.Position,
	logger *zap.Logger,
	collected *[]*schema.TreeNode,

) {
	if tree == nil {
		return
	}

	logger.Debug(
		"collectElementsOnLine: checking element",
		zap.String("path", tree.Path),
		zap.Any("range", tree.Range),
		zap.Any("pos", pos),
	)
	if lineContainsLSPPoint(tree.Range, pos) {
		logger.Debug(
			"collectElementsOnLine: found element",
			zap.String("path", tree.Path),
			zap.Any("range", tree.Range),
		)
		logger.Debug("Children length", zap.Int("length", len(tree.Children)))
		*collected = append(*collected, tree)
		i := 0
		for i < len(tree.Children) {
			collectElementsOnLine(tree.Children[i], pos, logger, collected)
			i += 1
		}
	}
}

func findNodeByPath(
	tree *schema.TreeNode,
	path string,
	logger *zap.Logger,
) *schema.TreeNode {
	if tree == nil {
		logger.Debug("findNodeByPath: tree is nil")
		return nil
	}

	if tree.Path == path {
		logger.Debug("findNodeByPath: found node", zap.String("path", path))
		return tree
	}

	node := (*schema.TreeNode)(nil)
	i := 0
	for node == nil && i < len(tree.Children) {
		child := tree.Children[i]
		if strings.HasPrefix(path, child.Path) {
			logger.Debug("findNodeByPath: found child matching prefix", zap.String("path", child.Path))
			node = findNodeByPath(tree.Children[i], path, logger)
		} else {
			logger.Debug("findNodeByPath: child does not match prefix", zap.String("path", child.Path))
		}
		i += 1
	}

	return node
}
