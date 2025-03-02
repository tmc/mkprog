package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

//go:embed system-prompt.txt
var systemPrompt string

// ToolCategory represents a category of tools in the mkprog toolchain
type ToolCategory string

const (
	CategoryGeneration  ToolCategory = "Code Generation"
	CategoryImprovement ToolCategory = "Code Improvement"
	CategoryAnalysis    ToolCategory = "Planning & Analysis"
	CategoryGit         ToolCategory = "Git Integration"
	CategoryMeta        ToolCategory = "Meta Tools"
)

// existingTools represents a mapping of tool names to their categories and descriptions
var existingTools = map[string]struct {
	Category    ToolCategory
	Description string
}{
	"mkprog":         {CategoryGeneration, "Generate complete Go programs from natural language descriptions"},
	"better-mkprog":  {CategoryGeneration, "Enhanced code generation with more advanced patterns"},
	"mkprogctx":      {CategoryGeneration, "Context-aware code generation"},
	"refactor":       {CategoryImprovement, "AI-assisted Go code refactoring with pattern recognition"},
	"fixme":          {CategoryImprovement, "Fix common issues in Go code"},
	"fixprog":        {CategoryImprovement, "Comprehensive codebase issue resolution"},
	"improveprog":    {CategoryImprovement, "General code improvement suggestions"},
	"modprog":        {CategoryImprovement, "Modify code based on specific requirements"},
	"planprog":       {CategoryAnalysis, "Create detailed implementation plans for complex features"},
	"depvis":         {CategoryAnalysis, "Go dependency visualization and analysis"},
	"benchviz":       {CategoryAnalysis, "Parse and visualize Go benchmark results"},
	"token-tree":     {CategoryAnalysis, "Visualize token usage across a codebase"},
	"askprog":        {CategoryAnalysis, "Ask questions about code and get detailed answers"},
	"auto-git-commit": {CategoryGit, "Automatic commit message generation"},
	"git-commit-style": {CategoryGit, "Analysis and standardization of commit messages"},
	"list-tools":     {CategoryMeta, "List all available tools with descriptions"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Command-line flags
	outputFile := pflag.StringP("output", "o", "", "Output file for the plan (defaults to stdout)")
	category := pflag.StringP("category", "c", "", "Tool category (generation, analysis, improvement, git, meta)")
	interactive := pflag.BoolP("interactive", "i", false, "Run in interactive mode for plan refinement")
	exampleCount := pflag.IntP("examples", "e", 2, "Number of example use cases to generate")
	showExisting := pflag.BoolP("existing", "x", false, "List existing tools that could be referenced in the plan")
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose output")
	pflag.Parse()

	// Show existing tools if requested
	if *showExisting {
		fmt.Println("Existing tools in the mkprog toolchain:")
		fmt.Println()
		fmt.Printf("%-20s %-20s %s\n", "NAME", "CATEGORY", "DESCRIPTION")
		fmt.Printf("%-20s %-20s %s\n", "----", "--------", "-----------")
		
		// Group by category
		toolsByCategory := make(map[ToolCategory][]string)
		for name, info := range existingTools {
			toolsByCategory[info.Category] = append(toolsByCategory[info.Category], name)
		}
		
		// Print each category
		for _, cat := range []ToolCategory{
			CategoryGeneration, CategoryImprovement, CategoryAnalysis, CategoryGit, CategoryMeta,
		} {
			for _, name := range toolsByCategory[cat] {
				info := existingTools[name]
				fmt.Printf("%-20s %-20s %s\n", name, string(info.Category), info.Description)
			}
		}
		return nil
	}

	// Get tool description from args or prompt
	var toolDescription string
	if pflag.NArg() > 0 {
		toolDescription = strings.Join(pflag.Args(), " ")
	} else {
		fmt.Println("Please provide a brief description of the tool you want to plan:")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				break
			}
			toolDescription += line + "\n"
		}
		toolDescription = strings.TrimSpace(toolDescription)
	}

	if toolDescription == "" {
		return fmt.Errorf("no tool description provided")
	}

	// Set up the category instruction
	var categoryInstruction string
	if *category != "" {
		categoryMap := map[string]ToolCategory{
			"generation":  CategoryGeneration,
			"improvement": CategoryImprovement,
			"analysis":    CategoryAnalysis,
			"git":         CategoryGit,
			"meta":        CategoryMeta,
		}
		
		if cat, ok := categoryMap[strings.ToLower(*category)]; ok {
			categoryInstruction = fmt.Sprintf("Create a tool in the %s category.", cat)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Unknown category '%s'. Using default.\n", *category)
		}
	}

	// Setup examples instruction
	examplesInstruction := fmt.Sprintf("Include exactly %d concrete use case examples.", *exampleCount)

	// Create the client
	ctx := context.Background()
	client, err := anthropic.New(
		anthropic.WithAnthropicBetaHeader(anthropic.MaxTokensAnthropicSonnet35),
	)
	if err != nil {
		return fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	// Prepare the input
	prompt := fmt.Sprintf(`Design a new unix-style command-line tool for the mkprog toolchain based on this description:

"%s"

%s
%s

Please provide a complete and detailed specification according to the required output format.`, 
		toolDescription, categoryInstruction, examplesInstruction)

	if *verbose {
		fmt.Fprintf(os.Stderr, "Generating tool specification...\n")
	}

	// Generate the plan
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	// Generate initial plan
	resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}

	plan := llms.MessageContentToString(resp)

	// Interactive mode
	if *interactive {
		plan = interactiveRefinement(ctx, client, prompt, plan)
	}

	// Write the output
	var out io.Writer = os.Stdout
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		out = file
	}

	// Add header with metadata
	fmt.Fprintf(out, "# Tool Specification\n\n")
	fmt.Fprintf(out, "- **Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(out, "- **Description:** %s\n\n", toolDescription)
	
	// Write the plan
	fmt.Fprintln(out, plan)

	// Additional information for tool creation
	if *outputFile != "" && *verbose {
		suggestNextSteps(*outputFile)
	}

	return nil
}

// interactiveRefinement allows the user to refine the plan interactively
func interactiveRefinement(ctx context.Context, client llms.Model, initialPrompt, initialPlan string) string {
	plan := initialPlan
	
	fmt.Println("\n=== Initial Tool Specification ===\n")
	fmt.Println(plan)
	
	scanner := bufio.NewScanner(os.Stdin)
	iteration := 1
	
	for {
		fmt.Println("\n=== Refinement Options ===")
		fmt.Println("1: Add a specific feature")
		fmt.Println("2: Modify the implementation plan")
		fmt.Println("3: Change integration points")
		fmt.Println("4: Adjust example use cases")
		fmt.Println("5: Done - save specification")
		fmt.Print("\nEnter choice (1-5): ")
		
		scanner.Scan()
		choice := scanner.Text()
		
		if choice == "5" {
			break
		}
		
		var refinementPrompt string
		switch choice {
		case "1":
			fmt.Print("Describe the feature to add: ")
			scanner.Scan()
			feature := scanner.Text()
			refinementPrompt = fmt.Sprintf("Please add the following feature to the tool specification: %s", feature)
		case "2":
			fmt.Print("Describe how to modify the implementation plan: ")
			scanner.Scan()
			implementation := scanner.Text()
			refinementPrompt = fmt.Sprintf("Please modify the implementation plan as follows: %s", implementation)
		case "3":
			fmt.Print("Describe how to change the integration points: ")
			scanner.Scan()
			integration := scanner.Text()
			refinementPrompt = fmt.Sprintf("Please modify the integration points as follows: %s", integration)
		case "4":
			fmt.Print("Describe how to adjust the example use cases: ")
			scanner.Scan()
			examples := scanner.Text()
			refinementPrompt = fmt.Sprintf("Please adjust the example use cases as follows: %s", examples)
		default:
			fmt.Println("Invalid choice. Please try again.")
			continue
		}
		
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, initialPrompt),
			llms.TextParts(llms.ChatMessageTypeAssistant, plan),
			llms.TextParts(llms.ChatMessageTypeHuman, refinementPrompt),
		}
		
		fmt.Fprintf(os.Stderr, "Refining plan (iteration %d)...\n", iteration)
		resp, err := client.GenerateContent(ctx, messages, llms.WithTemperature(0.1), llms.WithMaxTokens(4000))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error refining plan: %v\n", err)
			continue
		}
		
		plan = llms.MessageContentToString(resp)
		fmt.Println("\n=== Refined Tool Specification ===\n")
		fmt.Println(plan)
		
		iteration++
	}
	
	return plan
}

// suggestNextSteps suggests the next steps to create the tool
func suggestNextSteps(planFile string) {
	toolName := "new-tool" // Default name
	
	// Try to extract the tool name from the plan
	file, err := os.Open(planFile)
	if err == nil {
		defer file.Close()
		
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "- Name:") || strings.HasPrefix(line, "## Name:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					extractedName := strings.TrimSpace(parts[1])
					// If the name contains spaces or other characters, clean it up
					extractedName = strings.ToLower(extractedName)
					extractedName = strings.ReplaceAll(extractedName, " ", "-")
					if extractedName != "" {
						toolName = extractedName
						break
					}
				}
			}
		}
	}
	
	fmt.Println("\nNext steps to create this tool:")
	fmt.Printf("1. Create tool directory: mkdir -p tools/%s\n", toolName)
	fmt.Printf("2. Generate code: mkprog tools/%s \"$(cat %s | grep -v '^#')\"\n", toolName, planFile)
	fmt.Printf("3. Improve the code: improveprog tools/%s\n", toolName)
	fmt.Printf("4. Build the tool: (cd tools/%s && go build)\n", toolName)
	fmt.Printf("5. Test the tool: (cd tools/%s && go test)\n", toolName)
	fmt.Println("6. Add to toolchain: git add tools/" + toolName)
	fmt.Println("7. Commit your changes: git commit -m \"Add " + toolName + " tool\"")
}

// guessToolName attempts to guess a reasonable name for the tool based on its description
func guessToolName(description string) string {
	// Start by looking for common tool name patterns
	prefixes := []string{"go", "gen", "mk", "create", "build", "analyze", "fix", "improve", "convert", "transform"}
	
	for _, prefix := range prefixes {
		// Look for prefix-word pattern
		pattern := prefix + "[a-z]+"
		re := strings.NewReplacer(" ", "", "-", "", "_", "")
		clean := re.Replace(strings.ToLower(description))
		
		// Find words that start with the prefix
		for _, word := range strings.Fields(clean) {
			if strings.HasPrefix(word, prefix) && len(word) > len(prefix) {
				return word
			}
		}
	}
	
	// Fall back to using key words from the description
	words := strings.Fields(strings.ToLower(description))
	for _, word := range words {
		// Skip common short words
		if len(word) > 3 && !strings.Contains("the and for with that from this tool program code", word) {
			return word + "-tool"
		}
	}
	
	return "new-tool"
}