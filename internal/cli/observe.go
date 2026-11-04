package cli

// R: expose additive read-only observation with explicit selection and witness limits.

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/mako10k/sealgraph/internal/repository"
)

func runObserve(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return flagUsageError(stderr, "observe", err)
	}
	options, err := parseObserveArgs(args)
	if err != nil {
		return flagUsageError(stderr, "observe", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "observe", err)
	}
	result, err := repo.Observe(ctx, options)
	if err != nil {
		return commandError(stderr, "observe", err)
	}
	var rendered bytes.Buffer
	if output.JSON {
		encoder := json.NewEncoder(&rendered)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(result); err != nil {
			return commandError(stderr, "observe", err)
		}
		return writeOperatorOutput(stdout, stderr, "observe", literalizeInspectionUnicodeSeparators(rendered.Bytes()))
	}
	printObserveHuman(&rendered, result)
	return writeOperatorOutput(stdout, stderr, "observe", rendered.Bytes())
}

func parseObserveArgs(args []string) (repository.ObservationOptions, error) {
	result := repository.ObservationOptions{MaxPaths: 100}
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		result.Selector, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("observe", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var max singleString
	flags.Var(&max, "max-paths", "positive witness bound per reason")
	flags.BoolVar(&result.Inventory, "inventory", false, "include retained physical Seals")
	if err := flags.Parse(args); err != nil {
		return result, err
	}
	if flags.NArg() > 1 || result.Selector != "" && flags.NArg() != 0 {
		return result, fmt.Errorf("observe accepts at most one SELECTOR")
	}
	if flags.NArg() == 1 {
		result.Selector = flags.Arg(0)
	}
	if result.Inventory && result.Selector != "" {
		return result, fmt.Errorf("--inventory conflicts with SELECTOR; choose current selection or retained inventory")
	}
	if max.set {
		n, err := strconv.Atoi(max.value)
		if err != nil || n < 1 || n > 10000 {
			return result, fmt.Errorf("--max-paths must be between 1 and 10000")
		}
		result.MaxPaths = n
	}
	return result, nil
}
