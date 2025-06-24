package languageservices

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// SignatureService is a service that provides functionality
// for function signature help.
type SignatureService struct {
	funcRegistry provider.FunctionRegistry
	logger       *zap.Logger
}

// NewSignatureService creates a new service for function signature help.
func NewSignatureService(
	funcRegistry provider.FunctionRegistry,
	logger *zap.Logger,
) *SignatureService {
	return &SignatureService{
		funcRegistry,
		logger,
	}
}

// GetFunctionSignatures returns the function signatures for the given
// blueprint and signature help parameters including the current position
// in the source document.
func (s *SignatureService) GetFunctionSignatures(
	ctx *common.LSPContext,
	tree *schema.TreeNode,
	params *lsp.TextDocumentPositionParams,
) ([]*lsp.SignatureInformation, error) {
	s.logger.Debug("Searching for function at position", zap.Any("position", params.Position))
	subFunc := findFunctionAtPosition(tree, params.Position, s.logger)
	if subFunc == nil {
		return []*lsp.SignatureInformation{}, nil
	}

	return s.SignatureInfoFromFunction(subFunc, ctx)
}

func (s *SignatureService) SignatureInfoFromFunction(
	subFunc *substitutions.SubstitutionFunctionExpr,
	ctx *common.LSPContext,
) ([]*lsp.SignatureInformation, error) {
	defOutput, err := s.funcRegistry.GetDefinition(
		ctx.Context,
		string(subFunc.FunctionName),
		&provider.FunctionGetDefinitionInput{},
	)
	if err != nil {
		s.logger.Error("Failed to get function definition", zap.Error(err))
		return []*lsp.SignatureInformation{}, nil
	}

	if defOutput.Definition == nil {
		return []*lsp.SignatureInformation{}, nil
	}

	paramLabels := createParamLabels(defOutput.Definition)
	sigLabel := createFunctionSignatureLabel(
		string(subFunc.FunctionName),
		paramLabels,
		defOutput.Definition.Return,
	)

	return []*lsp.SignatureInformation{
		{
			Label:         sigLabel,
			Documentation: createFuncDocumentation(defOutput.Definition),
			Parameters:    createLSPParams(paramLabels, defOutput.Definition.Parameters),
		},
	}, nil
}

func findFunctionAtPosition(
	tree *schema.TreeNode,
	pos lsp.Position,
	logger *zap.Logger,
) *substitutions.SubstitutionFunctionExpr {
	if tree == nil {
		return nil
	}

	subFunc := (*substitutions.SubstitutionFunctionExpr)(nil)
	if containsLSPPoint(tree.Range, pos, 0 /* columnLeeway */) {
		var isParentSubFunc bool
		subFunc, isParentSubFunc = tree.SchemaElement.(*substitutions.SubstitutionFunctionExpr)
		if isParentSubFunc && len(tree.Children) == 0 {
			return subFunc
		}

		i := 0
		subFuncFromChildren := (*substitutions.SubstitutionFunctionExpr)(nil)
		for subFuncFromChildren == nil && i < len(tree.Children) {
			subFuncFromChildren = findFunctionAtPosition(tree.Children[i], pos, logger)
			i += 1
		}

		if subFuncFromChildren != nil {
			subFunc = subFuncFromChildren
		}
	}

	return subFunc
}

func createFuncDocumentation(def *function.Definition) any {
	markdown := def.FormattedDescription
	if markdown != "" {
		return lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: markdown,
		}
	}

	return def.Description
}

func createLSPParams(labels []string, params []function.Parameter) []*lsp.ParameterInformation {
	lspParams := make([]*lsp.ParameterInformation, len(params))
	for i, param := range params {
		lspParams[i] = &lsp.ParameterInformation{
			Label:         labels[i],
			Documentation: createParamDocumentation(param),
		}
	}
	return lspParams
}

func createParamDocumentation(param function.Parameter) any {
	markdown := param.GetFormattedDescription()
	if markdown != "" {
		return lsp.MarkupContent{
			Kind:  lsp.MarkupKindMarkdown,
			Value: markdown,
		}
	}

	return param.GetDescription()
}

func createFunctionSignatureLabel(name string, paramLabels []string, defReturn function.Return) string {
	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteString("(")
	for i, label := range paramLabels {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(label)
	}

	sb.WriteString(")")
	sb.WriteString(" -> ")
	sb.WriteString(renderReturnType(defReturn))

	return sb.String()
}

func createParamLabels(definition *function.Definition) []string {
	labels := make([]string, len(definition.Parameters))
	for i, param := range definition.Parameters {
		labels[i] = fmt.Sprintf("%s: %s", param.GetLabel(), renderParameterType(param))
	}
	return labels
}

func renderParameterType(param function.Parameter) string {
	switch param := param.(type) {
	case *function.ScalarParameter:
		return string(param.GetType())
	case *function.ListParameter:
		elementType := getTypeDefinitionLabel(param.ElementType)
		return fmt.Sprintf("list[%s]", elementType)
	case *function.MapParameter:
		valueType := getTypeDefinitionLabel(param.ElementType)
		return fmt.Sprintf("map[string, %s]", valueType)
	case *function.ObjectParameter:
		return "object"
	case *function.AnyParameter:
		return "any"
	case *function.FunctionParameter:
		return "function"
	case *function.VariadicParameter:
		if !param.SingleType {
			return "any..."
		}
		elementType := getTypeDefinitionLabel(param.Type)
		return fmt.Sprintf("%s...", elementType)
	default:
		return "unknown"
	}
}

func renderReturnType(defReturn function.Return) string {
	switch defReturn := defReturn.(type) {
	case *function.ScalarReturn:
		return string(defReturn.GetType())
	case *function.ListReturn:
		elementType := getTypeDefinitionLabel(defReturn.ElementType)
		return fmt.Sprintf("list[%s]", elementType)
	case *function.MapReturn:
		valueType := getTypeDefinitionLabel(defReturn.ElementType)
		return fmt.Sprintf("map[string, %s]", valueType)
	case *function.ObjectReturn:
		return "object"
	case *function.AnyReturn:
		return "any"
	case *function.FunctionReturn:
		return "function"
	default:
		return "unknown"
	}
}

func getTypeDefinitionLabel(def function.ValueTypeDefinition) string {
	switch def := def.(type) {
	case *function.ValueTypeDefinitionScalar:
		return string(def.Type)
	case *function.ValueTypeDefinitionList:
		elementType := getTypeDefinitionLabel(def.ElementType)
		return fmt.Sprintf("list[%s]", elementType)
	case *function.ValueTypeDefinitionMap:
		valueType := getTypeDefinitionLabel(def.ElementType)
		return fmt.Sprintf("map[string, %s]", valueType)
	case *function.ValueTypeDefinitionObject:
		return "object"
	case *function.ValueTypeDefinitionAny:
		return "any"
	case *function.ValueTypeDefinitionFunction:
		return "function"
	default:
		return "unknown"
	}
}
