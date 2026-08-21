package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/repository"
)

func runBashCompletion(workDir string, words []string, stdout io.Writer) int {
	mode, values := bashCompletion(workDir, words)
	fmt.Fprintf(stdout, "__sealgraph_completion_mode=%s\n", mode)
	for _, value := range values {
		fmt.Fprintln(stdout, value)
	}
	return 0
}

func bashCompletion(workDir string, words []string) (string, []string) {
	current := ""
	if len(words) != 0 {
		current = words[len(words)-1]
	}
	prior := words[:max(0, len(words)-1)]
	if len(prior) != 0 {
		switch prior[len(prior)-1] {
		case "--file", "--content-file":
			return "file", nil
		case "--format":
			return "plain", []string{"human", "json"}
		}
	}
	if len(prior) == 0 {
		return "plain", topLevelCompletionValues()
	}
	if prior[0] == "help" {
		return "plain", helpCompletionValues(prior[1:])
	}
	if len(prior) == 1 {
		if entry, ok := commandHelpRegistry[prior[0]]; ok && len(entry.Subcommands) != 0 {
			return "plain", append(append([]string{}, entry.Subcommands...), commandOptions(entry)...)
		}
	}
	path := prior[0]
	if len(prior) > 1 {
		if _, ok := commandHelpRegistry[path+" "+prior[1]]; ok {
			path += " " + prior[1]
		}
	}
	if strings.HasPrefix(current, "-") {
		if entry, ok := commandHelpRegistry[path]; ok {
			return "plain", commandOptions(entry)
		}
	}
	return "plain", repositoryCompletionValues(workDir, path)
}

func topLevelCompletionValues() []string {
	values := []string{"help", "version", "--help", "--version"}
	for path := range commandHelpRegistry {
		if !strings.Contains(path, " ") {
			values = append(values, path)
		}
	}
	sort.Strings(values)
	return values
}

func helpCompletionValues(path []string) []string {
	if len(path) == 0 {
		values := append(topLevelCompletionValues(), "concepts", "selectors", "usecases")
		sort.Strings(values)
		return values
	}
	entry, ok := commandHelpRegistry[strings.Join(path, " ")]
	if !ok {
		return nil
	}
	return append([]string{}, entry.Subcommands...)
}

func commandOptions(entry commandHelp) []string {
	values := make([]string, 0, len(entry.Options))
	for _, option := range entry.Options {
		name := strings.Fields(option.Syntax)[0]
		name = strings.Split(name, "|")[0]
		values = append(values, name)
	}
	sort.Strings(values)
	return values
}

func repositoryCompletionValues(workDir, path string) []string {
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return nil
	}
	names, err := repo.CompletionNames(context.Background())
	if err != nil {
		return nil
	}
	switch path {
	case "seal", "candidate show", "candidate compare", "candidate discard":
		return names.Candidates
	case "source show", "source compare", "source rebind", "source unbind":
		return names.Sources
	default:
		return names.REFs
	}
}
