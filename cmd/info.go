package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <app>",
	Short: "Show app information",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]
		cfg := config.Load()

		man, bucketName, err := findManifest(cfg, app)
		if err != nil {
			return err
		}

		urls, bins, extractDir, isInno := resolveManifest(man)

		fmt.Printf("%sName:%s     %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Bold, app, progress.Reset)
		fmt.Printf("%sBucket:%s   %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Yellow, bucketName, progress.Reset)
		fmt.Printf("%sVersion:%s  %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Green, man.Version, progress.Reset)
		if man.Description != "" {
			fmt.Printf("%sDescription:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, man.Description)
		}
		if man.Homepage != "" {
			fmt.Printf("%sHomepage:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, man.Homepage)
		}
		if man.License != nil {
			licStr := string(man.License)
			if strings.HasPrefix(licStr, "{") {
				var licObj struct {
					Identifier string `json:"identifier"`
					URL        string `json:"url"`
				}
				if err := unmarshalRaw(man.License, &licObj); err == nil && licObj.Identifier != "" {
					fmt.Printf("%sLicense:%s  %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Yellow, licObj.Identifier, progress.Reset)
				}
			} else {
				var lic string
				if err := unmarshalRaw(man.License, &lic); err == nil {
					fmt.Printf("%sLicense:%s  %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Yellow, lic, progress.Reset)
				}
			}
		}

		if extractDir != "" {
			fmt.Printf("%sExtract Dir:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, extractDir)
		}
		if isInno {
			fmt.Printf("%sInnoSetup:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Green+"true"+progress.Reset)
		}

		for _, u := range urls {
			fmt.Printf("%sURL (%d/%d):%s %s\n",
				progress.Cyan+progress.Bold,
				indexOf(&urls, &u)+1, len(urls),
				progress.Reset, u.URL)
		}

		if len(bins) > 0 {
			fmt.Printf("%sBin:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, strings.Join(bins, ", "))
		}
		if len(man.Depends) > 0 {
			fmt.Printf("%sDepends:%s %s%s%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Yellow, strings.Join(man.Depends, ", "), progress.Reset)
		}
		if len(man.Notes) > 0 {
			fmt.Printf("\n%sNotes:%s\n", progress.Cyan+progress.Bold, progress.Reset)
			for _, n := range man.Notes {
				fmt.Printf("  %s\n", n)
			}
		}
		if len(man.Suggestions) > 0 {
			fmt.Printf("%sSuggestions:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, strings.Join(man.Suggestions, ", "))
		}
		if len(man.Persist) > 0 {
			var persistStrs []string
			for _, p := range man.Persist {
				if p.Source == p.Target {
					persistStrs = append(persistStrs, p.Source)
				} else {
					persistStrs = append(persistStrs, fmt.Sprintf("%s -> %s", p.Source, p.Target))
				}
			}
			fmt.Printf("%sPersist:%s %s\n", progress.Cyan+progress.Bold, progress.Reset, strings.Join(persistStrs, ", "))
		}

		// Check if installed
		verDir := cfg.VersionDir(app, man.Version)
		if _, err := os.Stat(verDir); err == nil {
			fmt.Printf("\n%sStatus:%s %sinstalled%s\n", progress.Cyan+progress.Bold, progress.Reset, progress.Green, progress.Reset)
		}
		return nil
	},
}

func indexOf(urls *[]urlInfo, u *urlInfo) int {
	for i := range *urls {
		if &(*urls)[i] == u {
			return i
		}
	}
	return -1
}

func unmarshalRaw(raw json.RawMessage, v interface{}) error {
	return json.Unmarshal(raw, v)
}
