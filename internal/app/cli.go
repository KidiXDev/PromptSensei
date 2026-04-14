package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/logging"
	"github.com/kidixdev/PromptSensei/internal/tui"
)

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	if err := logging.Init(); err != nil {
		fmt.Fprintf(errOut, "warning: logger init failed: %v\n", err)
	} else {
		fmt.Fprintf(out, "debug logging enabled: %s\n", logging.Path())
		logging.Info("application start", "args", strings.Join(args, " "))
	}

	if len(args) >= 2 && args[0] == "config" && args[1] == "init" {
		paths, err := config.ResolvePaths()
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		if err := config.Bootstrap(paths); err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "initialized config at %s\n", paths.RootDir)
		return 0
	}

	runtime, notes, err := NewRuntime(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	for _, note := range notes {
		fmt.Fprintf(out, "info: %s\n", note)
	}

	isTUI := len(args) == 0 || args[0] == "tui"
	if !isTUI {
		rebuilt, reasons, err := runtime.Dataset.EnsureFresh(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "error: dataset bootstrap: %v\n", err)
			return 1
		}
		if rebuilt {
			fmt.Fprintf(out, "info: dataset rebuilt: %s\n", joinReasons(reasons))
		}
	}

	if isTUI {
		saveConfig := func(next config.Config) error {
			prev := runtime.Config
			if err := runtime.Providers.UpdateConfig(next.Provider); err != nil {
				return err
			}
			runtime.Dataset.UpdateConfig(next)
			runtime.Prompt.UpdateConfig(next)

			if err := config.Save(runtime.Paths, next); err != nil {
				_ = runtime.Providers.UpdateConfig(prev.Provider)
				runtime.Dataset.UpdateConfig(prev)
				runtime.Prompt.UpdateConfig(prev)
				return err
			}
			runtime.Config = next
			return nil
		}

		if err := tui.Run(ctx, runtime.Prompt, runtime.Dataset, runtime.Knowledge, runtime.Paths.ConfigFile, runtime.Config, saveConfig, in, out); err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		return 0
	}

	switch args[0] {
	case "enhance":
		return runEnhanceCommand(ctx, runtime, false, args[1:], out, errOut)
	case "create":
		return runEnhanceCommand(ctx, runtime, true, args[1:], out, errOut)
	case "dataset":
		return runDatasetCommand(ctx, runtime, args[1:], out, errOut)
	case "knowledge":
		return runKnowledgeCommand(runtime, args[1:], out, errOut)
	case "doctor":
		return runDoctorCommand(ctx, runtime, out, errOut)
	case "help", "--help", "-h":
		printHelp(out)
		return 0
	default:
		fmt.Fprintf(errOut, "error: unknown command %q\n", args[0])
		printHelp(out)
		return 1
	}
}

func runEnhanceCommand(ctx context.Context, runtime *Runtime, create bool, args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("enhance", flag.ContinueOnError)
	fs.SetOutput(errOut)
	prompt := fs.String("prompt", "", "prompt text")
	modeRaw := fs.String("mode", string(runtime.Config.General.DefaultMode), "mode: natural|booru|hybrid")
	knowledgeRaw := fs.String("knowledge", "", "comma-separated knowledge files")
	strict := fs.Bool("strict", false, "enable strict booru filtering")
	debug := fs.Bool("debug", false, "print retrieval debug info")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if strings.TrimSpace(*prompt) == "" {
		fmt.Fprintln(errOut, "error: --prompt is required")
		return 1
	}

	mode, err := domain.ParseMode(*modeRaw)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}

	result, warnings, err := runtime.Prompt.Enhance(ctx, domain.EnhanceRequest{
		Prompt:         *prompt,
		Mode:           mode,
		KnowledgeFiles: parseCommaList(*knowledgeRaw),
		StrictBooru:    *strict,
		CreateMode:     create,
	})
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	for _, warning := range warnings {
		fmt.Fprintf(errOut, "warning: %s\n", warning)
	}

	if *debug {
		fmt.Fprintf(out, "provider=%s used=%t strict=%t\n", result.ProviderName, result.UsedProvider, result.ValidationApplied)
		fmt.Fprintf(out, "retrieval: confirmed=%d character=%d suggested=%d rejected=%d\n",
			len(result.Retrieval.ConfirmedTags),
			len(result.Retrieval.CharacterTags),
			len(result.Retrieval.SuggestedTags),
			len(result.Retrieval.RejectedTags),
		)
	}
	fmt.Fprintln(out, result.Output)
	return 0
}

func runDatasetCommand(ctx context.Context, runtime *Runtime, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "error: dataset requires subcommand: rebuild|status")
		return 1
	}

	switch args[0] {
	case "rebuild":
		meta, err := runtime.Dataset.Rebuild(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "dataset rebuilt at %s\n", meta.LastRebuildUTC)
		fmt.Fprintf(out, "counts: tags=%d aliases=%d characters=%d triggers=%d core_tags=%d\n",
			meta.Counts.Tags, meta.Counts.TagAliases, meta.Counts.Characters, meta.Counts.CharacterTriggers, meta.Counts.CharacterCoreTags)
		return 0
	case "status":
		status, err := runtime.Dataset.Status(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "tag_csv=%s\n", status.Paths.TagCSV)
		fmt.Fprintf(out, "character_csv=%s\n", status.Paths.CharacterCSV)
		fmt.Fprintf(out, "cache=%s\n", status.Paths.DBPath)
		fmt.Fprintf(out, "metadata=%s\n", status.MetadataPath)
		fmt.Fprintf(out, "db_exists=%t rebuild_needed=%t\n", status.HasDatabase, status.RebuildNeeded)
		if len(status.RebuildReasons) > 0 {
			fmt.Fprintf(out, "reasons=%s\n", strings.Join(status.RebuildReasons, "; "))
		}
		fmt.Fprintf(out, "rows tags=%d aliases=%d characters=%d triggers=%d core_tags=%d\n",
			status.Counts.Tags, status.Counts.TagAliases, status.Counts.Characters, status.Counts.CharacterTriggers, status.Counts.CharacterCoreTags)
		return 0
	default:
		fmt.Fprintf(errOut, "error: unknown dataset subcommand %q\n", args[0])
		return 1
	}
}

func runKnowledgeCommand(runtime *Runtime, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprintln(errOut, "error: knowledge requires subcommand: list")
		return 1
	}
	files, err := runtime.Knowledge.List()
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	for _, f := range files {
		fmt.Fprintln(out, f)
	}
	return 0
}

func runDoctorCommand(ctx context.Context, runtime *Runtime, out io.Writer, errOut io.Writer) int {
	status, err := runtime.Dataset.Status(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "config_dir=%s\n", runtime.Paths.RootDir)
	fmt.Fprintf(out, "config_file=%s\n", runtime.Paths.ConfigFile)
	fmt.Fprintf(out, "provider_enabled=%t provider_name=%s model=%s\n",
		runtime.Config.Provider.Enabled, runtime.Config.Provider.Name, runtime.Config.Provider.Model)
	fmt.Fprintf(out, "dataset_db_exists=%t rebuild_needed=%t\n", status.HasDatabase, status.RebuildNeeded)
	fmt.Fprintf(out, "dataset_rows tags=%d aliases=%d characters=%d triggers=%d core_tags=%d\n",
		status.Counts.Tags, status.Counts.TagAliases, status.Counts.Characters, status.Counts.CharacterTriggers, status.Counts.CharacterCoreTags)
	if len(status.RebuildReasons) > 0 {
		fmt.Fprintf(out, "dataset_reasons=%s\n", strings.Join(status.RebuildReasons, "; "))
	}
	if runtime.Config.Provider.Enabled && strings.TrimSpace(runtime.Config.Provider.APIKey) == "" {
		fmt.Fprintln(out, "warning=provider enabled but api_key is empty")
	}
	return 0
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `PromptSensei commands:
  prompt-sensei tui
  prompt-sensei enhance --prompt "..."
  prompt-sensei create --prompt "..."
  prompt-sensei dataset rebuild
  prompt-sensei dataset status
  prompt-sensei knowledge list
  prompt-sensei config init
  prompt-sensei doctor`)
}

func parseCommaList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "up-to-date"
	}
	out := reasons[0]
	for i := 1; i < len(reasons); i++ {
		out += "; " + reasons[i]
	}
	return out
}
