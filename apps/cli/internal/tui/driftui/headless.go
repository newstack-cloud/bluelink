package driftui

import (
	"sort"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
)

// HeadlessDriftPrinter prints drift information in headless mode.
type HeadlessDriftPrinter struct {
	printer *headless.Printer
	context DriftContext
}

// NewHeadlessDriftPrinter creates a new headless drift printer.
func NewHeadlessDriftPrinter(printer *headless.Printer, context DriftContext) *HeadlessDriftPrinter {
	return &HeadlessDriftPrinter{
		printer: printer,
		context: context,
	}
}

// PrintDriftDetected prints the full drift detection output.
func (p *HeadlessDriftPrinter) PrintDriftDetected(result *container.ReconciliationCheckResult) {
	if result == nil || p.printer == nil {
		return
	}

	w := p.printer.Writer()
	w.PrintlnEmpty()
	w.Println("⚠ Drift detected - external changes conflict with deployment")
	w.PrintlnEmpty()

	// Build tree for child blueprints
	childTree := p.buildChildTree(result)

	// Print parent-level resources (ChildPath == "")
	parentResources := filterResourcesByChildPath(result.Resources, "")
	if len(parentResources) > 0 {
		w.Println("Resources with drift:")
		for i := range parentResources {
			p.printResourceDrift(&parentResources[i], 1)
		}
		w.PrintlnEmpty()
	}

	// Print parent-level links (ChildPath == "")
	parentLinks := filterLinksByChildPath(result.Links, "")
	if len(parentLinks) > 0 {
		w.Println("Links with drift:")
		for i := range parentLinks {
			p.printLinkDrift(&parentLinks[i], 1)
		}
		w.PrintlnEmpty()
	}

	// Print child blueprints hierarchically
	if len(childTree) > 0 {
		w.Println("Child blueprints with drift:")
		p.printChildNodes(childTree, 1)
		w.PrintlnEmpty()
	}

	// Print resolution hint
	p.printHint()
}

func (p *HeadlessDriftPrinter) printResourceDrift(r *container.ResourceReconcileResult, indent int) {
	w := p.printer.Writer()
	indentStr := strings.Repeat("  ", indent)
	fieldIndent := strings.Repeat("  ", indent+1)

	// Use appropriate icon for drift type
	icon := "⚠"
	if r.Type == container.ReconciliationTypeInterrupted {
		icon = "!"
	}

	driftType := HumanReadableDriftType(r.Type)
	w.Printf("%s%s %s  %s (%s)\n", indentStr, icon, driftType, r.ResourceName, r.ResourceType)

	// For interrupted resources, show status transition
	if r.Type == container.ReconciliationTypeInterrupted {
		w.Printf("%sStatus: %s → %s\n", fieldIndent, r.OldStatus, r.NewStatus)
		if r.ResourceExists {
			w.Printf("%sResource exists: Yes\n", fieldIndent)
		} else {
			w.Printf("%sResource exists: No (not found externally)\n", fieldIndent)
		}
	}

	if r.Changes != nil {
		for _, field := range r.Changes.ModifiedFields {
			prevValue := headless.FormatMappingNode(field.PrevValue)
			newValue := headless.FormatMappingNode(field.NewValue)
			w.Printf("%s± %s: %s → %s\n", fieldIndent, field.FieldPath, prevValue, newValue)
		}
		for _, field := range r.Changes.NewFields {
			w.Printf("%s+ %s: %s\n", fieldIndent, field.FieldPath, headless.FormatMappingNode(field.NewValue))
		}
		for _, fieldPath := range r.Changes.RemovedFields {
			w.Printf("%s- %s\n", fieldIndent, fieldPath)
		}
	} else if r.Type == container.ReconciliationTypeInterrupted && r.ExternalState != nil {
		// For interrupted resources without computed changes, show external state exists
		w.Printf("%sExternal state available (use interactive mode to view details)\n", fieldIndent)
	}
}

func (p *HeadlessDriftPrinter) printLinkDrift(l *container.LinkReconcileResult, indent int) {
	w := p.printer.Writer()
	indentStr := strings.Repeat("  ", indent)

	driftType := HumanReadableDriftType(l.Type)
	w.Printf("%s⚠ %s  %s\n", indentStr, driftType, l.LinkName)

	if len(l.LinkDataUpdates) > 0 {
		fieldIndent := strings.Repeat("  ", indent+1)
		for path := range l.LinkDataUpdates {
			w.Printf("%s± %s (link data affected)\n", fieldIndent, path)
		}
	}
}

func (p *HeadlessDriftPrinter) printChildNodes(nodes []*headlessChildNode, indent int) {
	for _, node := range nodes {
		p.printChildNode(node, indent)
	}
}

func (p *HeadlessDriftPrinter) printChildNode(node *headlessChildNode, indent int) {
	w := p.printer.Writer()
	indentStr := strings.Repeat("  ", indent)

	// Count drifted items in this subtree
	count := countDriftedItems(node)
	w.Printf("%s⚠ %s (%d drifted)\n", indentStr, node.name, count)

	// Print resources at this level
	for i := range node.resources {
		p.printResourceDrift(node.resources[i], indent+1)
	}

	// Print links at this level
	for i := range node.links {
		p.printLinkDrift(node.links[i], indent+1)
	}

	if len(node.children) > 0 {
		childNames := sortedMapKeys(node.children)
		for _, name := range childNames {
			p.printChildNode(node.children[name], indent+1)
		}
	}
}

func (p *HeadlessDriftPrinter) printHint() {
	w := p.printer.Writer()
	w.Println("To resolve:")
	w.Println("  1. Review external changes manually and update your blueprint")
	w.Printf("  2. Or re-run with %s\n", HintForContext(p.context))
	w.PrintlnEmpty()
}

// headlessChildNode is a tree node for organizing child blueprint drift.
type headlessChildNode struct {
	name      string
	fullPath  string
	resources []*container.ResourceReconcileResult
	links     []*container.LinkReconcileResult
	children  map[string]*headlessChildNode
}

func (p *HeadlessDriftPrinter) buildChildTree(
	result *container.ReconciliationCheckResult,
) []*headlessChildNode {
	root := &headlessChildNode{children: make(map[string]*headlessChildNode)}

	// Insert all resources with non-empty ChildPath into the tree
	for i := range result.Resources {
		r := &result.Resources[i]
		if r.ChildPath != "" {
			insertResourceIntoTree(root, r.ChildPath, r)
		}
	}

	// Insert all links with non-empty ChildPath into the tree
	for i := range result.Links {
		l := &result.Links[i]
		if l.ChildPath != "" {
			insertLinkIntoTree(root, l.ChildPath, l)
		}
	}

	// Convert to sorted slice
	return sortedChildNodes(root.children)
}

func insertResourceIntoTree(
	root *headlessChildNode,
	childPath string,
	r *container.ResourceReconcileResult,
) {
	node := getOrCreateNode(root, childPath)
	node.resources = append(node.resources, r)
}

func insertLinkIntoTree(
	root *headlessChildNode,
	childPath string,
	l *container.LinkReconcileResult,
) {
	node := getOrCreateNode(root, childPath)
	node.links = append(node.links, l)
}

func getOrCreateNode(root *headlessChildNode, childPath string) *headlessChildNode {
	segments := strings.Split(childPath, ".")
	current := root
	for i, segment := range segments {
		if current.children == nil {
			current.children = make(map[string]*headlessChildNode)
		}
		child, exists := current.children[segment]
		if !exists {
			child = &headlessChildNode{
				name:     segment,
				fullPath: strings.Join(segments[:i+1], "."),
				children: make(map[string]*headlessChildNode),
			}
			current.children[segment] = child
		}
		current = child
	}
	return current
}

func filterResourcesByChildPath(
	resources []container.ResourceReconcileResult,
	childPath string,
) []container.ResourceReconcileResult {
	var filtered []container.ResourceReconcileResult
	for _, r := range resources {
		if r.ChildPath == childPath {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterLinksByChildPath(
	links []container.LinkReconcileResult,
	childPath string,
) []container.LinkReconcileResult {
	var filtered []container.LinkReconcileResult
	for _, l := range links {
		if l.ChildPath == childPath {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func countDriftedItems(node *headlessChildNode) int {
	count := len(node.resources) + len(node.links)
	for _, child := range node.children {
		count += countDriftedItems(child)
	}
	return count
}

func sortedMapKeys(m map[string]*headlessChildNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedChildNodes(m map[string]*headlessChildNode) []*headlessChildNode {
	keys := sortedMapKeys(m)
	nodes := make([]*headlessChildNode, 0, len(keys))
	for _, k := range keys {
		nodes = append(nodes, m[k])
	}
	return nodes
}
