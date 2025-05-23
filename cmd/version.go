package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	versionDetailed bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Eolas",
	Long:  `Display the version of Eolas currently installed.
	
Use --detailed to show additional build information, dependencies, and features.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Eolas version: %s\n", version)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

		if versionDetailed {
			fmt.Println()
			showDetailedVersion()
		}
	},
}

// showDetailedVersion displays comprehensive version and build information
func showDetailedVersion() {
	fmt.Println("Build Information:")
	fmt.Println("==================")

	// Get build info
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" {
			fmt.Printf("Module version: %s\n", info.Main.Version)
		}

		// Show build settings
		fmt.Println("\nBuild Settings:")
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					fmt.Printf("Git commit: %s\n", setting.Value[:7])
				} else {
					fmt.Printf("Git commit: %s\n", setting.Value)
				}
			case "vcs.time":
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					fmt.Printf("Build time: %s\n", t.Format("2006-01-02 15:04:05 UTC"))
				}
			case "vcs.modified":
				if setting.Value == "true" {
					fmt.Printf("Git status: modified (uncommitted changes)\n")
				} else {
					fmt.Printf("Git status: clean\n")
				}
			}
		}

		// Show dependencies
		fmt.Println("\nDependencies:")
		keyDeps := []string{
			"github.com/spf13/cobra",
			"modernc.org/sqlite",
			"github.com/google/uuid",
		}

		for _, dep := range info.Deps {
			for _, keyDep := range keyDeps {
				if strings.Contains(dep.Path, keyDep) {
					version := dep.Version
					if dep.Replace != nil {
						version = fmt.Sprintf("%s => %s", version, dep.Replace.Version)
					}
					fmt.Printf("  %s: %s\n", dep.Path, version)
				}
			}
		}
	}

	// Show runtime information
	fmt.Println("\nRuntime Information:")
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("Memory usage: %d KB\n", memStats.Alloc/1024)

	// Show features
	fmt.Println("\nSupported Features:")
	fmt.Println("  ✓ File-based configuration storage")
	fmt.Println("  ✓ SQLite configuration storage with versioning")
	fmt.Println("  ✓ Security analysis (privileged containers, capabilities, host access)")
	fmt.Println("  ✓ Configuration comparison between versions")
	fmt.Println("  ✓ Timeline reports with trend analysis")
	fmt.Println("  ✓ HTML reports with responsive design")
	fmt.Println("  ✓ Data export (JSON, CSV)")
	fmt.Println("  ✓ Data migration between storage backends")

	// Show storage backends
	fmt.Println("\nStorage Backends:")
	fmt.Println("  file   - Simple file-based storage (default)")
	fmt.Println("  sqlite - Advanced SQLite storage with versioning and history")

	// Show output formats
	fmt.Println("\nOutput Formats:")
	fmt.Println("  text - Standard terminal output")
	fmt.Println("  html - Interactive HTML reports")
	fmt.Println("  json - Structured JSON export")
	fmt.Println("  csv  - CSV export for spreadsheet analysis")
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&versionDetailed, "detailed", false, "Show detailed build information, dependencies, and features")
}