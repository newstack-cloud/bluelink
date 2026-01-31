package pluginlistui

import (
	"sort"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins"
	"github.com/newstack-cloud/deploy-cli-sdk/headless"
)

func (m *MainModel) printHeadlessPlugins(pluginList []*plugins.InstalledPlugin) {
	if m.printer == nil {
		return
	}

	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println(m.buildTitle())
	w.DoubleSeparator(60)

	if len(pluginList) == 0 {
		w.Println("No plugins found.")
	}

	for _, p := range pluginList {
		printHeadlessPluginEntry(w, p)
	}

	w.PrintlnEmpty()
	w.DoubleSeparator(60)
	w.Printf("Total: %d plugin(s)\n", len(pluginList))
	w.PrintlnEmpty()
}

func printHeadlessPluginEntry(w *headless.PrefixedWriter, p *plugins.InstalledPlugin) {
	typeLabel := p.Type
	if typeLabel == "" {
		typeLabel = "unknown"
	}
	w.Printf("  %s [%s]\n", p.ID, typeLabel)
	w.Printf("    Version: %s\n", p.Version)
	w.Printf("    Registry: %s\n", p.RegistryHost)
	w.Printf("    Installed: %s\n", p.InstalledAt.Format("2006-01-02 15:04:05"))
	printHeadlessDependencies(w, p.Dependencies)
}

func printHeadlessDependencies(w *headless.PrefixedWriter, deps map[string]string) {
	if len(deps) == 0 {
		return
	}

	w.Println("    Dependencies:")
	depIDs := make([]string, 0, len(deps))
	for depID := range deps {
		depIDs = append(depIDs, depID)
	}
	sort.Strings(depIDs)

	for _, depID := range depIDs {
		depVersion := deps[depID]
		if depVersion != "" {
			w.Printf("      - %s@%s\n", depID, depVersion)
		} else {
			w.Printf("      - %s\n", depID)
		}
	}
}

func (m *MainModel) printHeadlessError(err error) {
	if m.printer == nil {
		return
	}
	w := m.printer.Writer()
	w.PrintlnEmpty()
	w.Println("ERR List plugins failed")
	w.PrintlnEmpty()
	w.Printf("  Error: %s\n", err.Error())
}

func (m *MainModel) dispatchHeadlessOutput(
	pluginList []*plugins.InstalledPlugin,
	err error,
) {
	if err != nil {
		m.printHeadlessError(err)
		return
	}
	m.printHeadlessPlugins(pluginList)
}
