package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Type        string `json:"type"`
	Flags       string `json:"flags"`
	UsesStdin   bool   `json:"uses_stdin"`
}

type Config struct {
	AdditionalDirs []string `json:"additional_dirs"`
}

var (
	cache      map[string]Tool
	cacheMutex sync.RWMutex
	config     Config
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	loadConfig()
	loadCache()

	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	if len(os.Args) < 2 {
		return listAllTools()
	}

	switch os.Args[1] {
	case "search":
		return searchCmd.Parse(os.Args[2:])
	case "info":
		return infoCmd.Parse(os.Args[2:])
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func loadConfig() {
	configFile := "config.json"
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Warning: Could not read config file: %v\n", err)
		return
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Warning: Could not parse config file: %v\n", err)
	}
}

func loadCache() {
	cacheFile := "tools_cache.json"
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		cache = make(map[string]Tool)
		return
	}

	err = json.Unmarshal(data, &cache)
	if err != nil {
		fmt.Printf("Warning: Could not parse cache file: %v\n", err)
		cache = make(map[string]Tool)
	}
}

func saveCache() {
	cacheFile := "tools_cache.json"
	data, err := json.Marshal(cache)
	if err != nil {
		fmt.Printf("Warning: Could not marshal cache: %v\n", err)
		return
	}

	err = ioutil.WriteFile(cacheFile, data, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not save cache file: %v\n", err)
	}
}

func listAllTools() error {
	tools, err := scanTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		fmt.Printf("%s - %s\n", tool.Name, tool.Description)
	}

	return nil
}

func scanTools() ([]Tool, error) {
	var tools []Tool
	var wg sync.WaitGroup
	toolsChan := make(chan Tool)
	errorsChan := make(chan error)

	// Scan current repo
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanDirectory(".", toolsChan, errorsChan)
	}()

	// Scan PATH entries within user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %v", err)
	}

	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range pathDirs {
		if strings.HasPrefix(dir, homeDir) {
			wg.Add(1)
			go func(d string) {
				defer wg.Done()
				scanDirectory(d, toolsChan, errorsChan)
			}(dir)
		}
	}

	// Scan additional directories from config
	for _, dir := range config.AdditionalDirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			scanDirectory(d, toolsChan, errorsChan)
		}(dir)
	}

	go func() {
		wg.Wait()
		close(toolsChan)
		close(errorsChan)
	}()

	for {
		select {
		case tool, ok := <-toolsChan:
			if !ok {
				saveCache()
				return tools, nil
			}
			tools = append(tools, tool)
		case err := <-errorsChan:
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
}

func scanDirectory(dir string, toolsChan chan<- Tool, errorsChan chan<- error) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errorsChan <- fmt.Errorf("error accessing %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if info.Mode().Perm()&0111 != 0 {
			tool, err := getToolInfo(path)
			if err != nil {
				errorsChan <- fmt.Errorf("error getting tool info for %s: %v", path, err)
				return nil
			}
			toolsChan <- tool
		}

		return nil
	})

	if err != nil {
		errorsChan <- fmt.Errorf("error walking directory %s: %v", dir, err)
	}
}

func getToolInfo(path string) (Tool, error) {
	cacheMutex.RLock()
	cachedTool, exists := cache[path]
	cacheMutex.RUnlock()

	if exists {
		return cachedTool, nil
	}

	name := filepath.Base(path)
	toolType := "custom toolchain tool"
	if strings.HasPrefix(path, "/usr/bin") || strings.HasPrefix(path, "/bin") {
		toolType = "standard system tool"
	}

	description := getToolDescription(path)
	flags, usesStdin := getToolFlagsAndStdin(path)

	tool := Tool{
		Name:        name,
		Description: description,
		Location:    path,
		Type:        toolType,
		Flags:       flags,
		UsesStdin:   usesStdin,
	}

	cacheMutex.Lock()
	cache[path] = tool
	cacheMutex.Unlock()

	return tool, nil
}

func getToolDescription(path string) string {
	cmd := exec.Command(path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "No description available"
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "description") || strings.Contains(line, "usage") {
			return strings.TrimSpace(line)
		}
	}

	return "No description available"
}

func getToolFlagsAndStdin(path string) (string, bool) {
	cmd := exec.Command(path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}

	flags := ""
	usesStdin := false

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "-") || strings.Contains(line, "--") {
			flags += line + "\n"
		}
		if strings.Contains(strings.ToLower(line), "stdin") {
			usesStdin = true
		}
	}

	return strings.TrimSpace(flags), usesStdin
}

func searchTools(term string) error {
	tools, err := scanTools()
	if err != nil {
		return err
	}

	regex, err := regexp.Compile(strings.ToLower(term))
	if err != nil {
		return fmt.Errorf("invalid search term: %v", err)
	}

	for _, tool := range tools {
		if regex.MatchString(strings.ToLower(tool.Name)) || regex.MatchString(strings.ToLower(tool.Description)) {
			fmt.Printf("%s - %s\n", tool.Name, tool.Description)
		}
	}

	return nil
}

func displayToolInfo(name string) error {
	tools, err := scanTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		if tool.Name == name {
			fmt.Printf("Name: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Printf("Location: %s\n", tool.Location)
			fmt.Printf("Type: %s\n", tool.Type)
			fmt.Printf("Flags:\n%s\n", tool.Flags)
			fmt.Printf("Uses stdin: %v\n", tool.UsesStdin)
			return nil
		}
	}

	return fmt.Errorf("tool not found: %s", name)
}
