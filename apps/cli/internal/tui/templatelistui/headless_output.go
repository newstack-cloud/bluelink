package templatelistui

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/templates"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
)

func (m *MainModel) printHeadlessTemplates(tmpls []templates.Template) {
	if m.printer == nil {
		return
	}

	w := m.printer.Writer()
	w.PrintlnEmpty()

	title := "Available Templates"
	if m.searchTerm != "" {
		title = fmt.Sprintf("Available Templates (search: %q)", m.searchTerm)
	}
	w.Println(title)
	w.DoubleSeparator(60)

	if len(tmpls) == 0 {
		w.Println("No templates found.")
	} else {
		for _, t := range tmpls {
			printHeadlessTemplateEntry(w, t)
		}
	}

	w.PrintlnEmpty()
	w.DoubleSeparator(60)
	w.Printf("Total: %d template(s)\n", len(tmpls))
	w.PrintlnEmpty()
}

func printHeadlessTemplateEntry(w *headless.PrefixedWriter, t templates.Template) {
	w.Printf("  %s (%s)\n", t.Label, t.Key)
	w.Printf("    %s\n", t.Description)
	w.PrintlnEmpty()
}

func (m *MainModel) printHeadlessError(err error) {
	if m.printer == nil {
		return
	}
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR List templates failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *MainModel) dispatchHeadlessOutput(
	tmpls []templates.Template,
	err error,
) {
	if err != nil {
		m.printHeadlessError(err)
		return
	}
	m.printHeadlessTemplates(tmpls)
}
