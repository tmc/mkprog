package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	startCommit string
	endCommit   string
	dryRun      bool
	aiAssist    bool
	verbose     bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "backport-changes",
	Short: "Backport non-machine-made changes throughout git history",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&startCommit, "start-commit", "", "Earliest commit to start backporting (default: first commit)")
	rootCmd.Flags().StringVar(&endCommit, "end-commit", "HEAD", "Latest commit to backport to")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making actual changes")
	rootCmd.Flags().BoolVar(&aiAssist, "ai-assist", false, "Use AI to help resolve conflicts (requires OpenAI API key)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Provide detailed output of the backporting process")
}

func run() error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	if endCommit == "HEAD" {
		endCommit = head.Hash().String()
	}

	if startCommit == "" {
		startCommit, err = getFirstCommit(repo)
		if err != nil {
			return fmt.Errorf("failed to get first commit: %w", err)
		}
	}

	log.Printf("Backporting changes from %s to %s", startCommit, endCommit)

	if dryRun {
		log.Println("Dry run mode: No changes will be made")
	}

	changes, err := identifyNonMachineChanges(repo, startCommit, endCommit)
	if err != nil {
		return fmt.Errorf("failed to identify non-machine changes: %w", err)
	}

	if err := createBackportBranch(repo); err != nil {
		return fmt.Errorf("failed to create backport branch: %w", err)
	}

	if err := applyChanges(repo, changes); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	log.Println("Backporting completed successfully")
	return nil
}

func getFirstCommit(repo *git.Repository) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return "", err
	}

	var lastCommit *object.Commit
	err = cIter.ForEach(func(c *object.Commit) error {
		lastCommit = c
		return nil
	})

	if lastCommit == nil {
		return "", fmt.Errorf("no commits found")
	}

	return lastCommit.Hash.String(), nil
}

func identifyNonMachineChanges(repo *git.Repository, start, end string) ([]string, error) {
	log.Println("Identifying non-machine-made changes...")
	// Implementation omitted for brevity
	return []string{"change1", "change2"}, nil
}

func createBackportBranch(repo *git.Repository) error {
	log.Println("Creating backport branch...")
	if dryRun {
		return nil
	}
	// Implementation omitted for brevity
	return nil
}

func applyChanges(repo *git.Repository, changes []string) error {
	log.Println("Applying changes...")
	if dryRun {
		return nil
	}
	// Implementation omitted for brevity
	return nil
}

func resolveConflict(conflict string) (string, error) {
	if !aiAssist {
		return "", fmt.Errorf("AI assistance is required to resolve conflicts")
	}

	client, err := openai.New()
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	ctx := context.Background()
	prompt := fmt.Sprintf("Resolve the following Git conflict:\n\n%s", conflict)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant that helps resolve Git conflicts."),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(1000))
	if err != nil {
		return "", fmt.Errorf("failed to generate AI response: %w", err)
	}

	resolution := resp.Choices[0].Content
	return extractResolution(resolution), nil
}

func extractResolution(aiResponse string) string {
	re := regexp.MustCompile("(?s)```.*?\n(.*?)```")
	matches := re.FindStringSubmatch(aiResponse)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return aiResponse
}
