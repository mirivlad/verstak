package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/verstak/verstak-desktop/internal/api"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	// ─── Initialize Core Registries ──────────────────────────
	capRegistry := capability.NewRegistry()
	contribRegistry := contribution.NewRegistry()
	permRegistry := permissions.NewRegistry()
	eventBus := events.NewBus()

	// ─── Plugin Discovery ───────────────────────────────────
	discoveryDirs := []string{
		"~/.config/verstak/plugins",
		"./plugins",
	}

	plugins, discErrors := plugin.DiscoverPlugins(discoveryDirs)
	for _, err := range discErrors {
		log.Printf("[plugin] discovery warning: %v", err)
	}

	log.Printf("[plugin] discovered %d plugins", len(plugins))

	// Create the App struct
	app := api.NewApp(capRegistry, contribRegistry, permRegistry, eventBus, plugins)

	// ─── Wails App ───────────────────────────────────────────
	err := wails.Run(&options.App{
		Title:            "Verstak",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		WindowStartState: options.Normal,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
