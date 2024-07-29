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
	"strings"
	"sync"
	"time"
)

type Tool struct {
	Name        string
	Description string
	Path        string
}

type Config struct {
	AdditionalDirectories []string `json:"additionalDirectories"`
}

var (
	cacheFile     = filepath.Join(os.TempDir(), "list-tools-cache.json")
	configFile    = filepath.Join(os.Getenv("HOME"), ".list-tools.json")
	cacheDuration = 24 * time.Hour
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	searchTerm := flag.String("search", "", "Search term for filtering tools")
	infoTool := flag.String("info", "", "Display detailed information about a specific tool")
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tools, err := getTools(config.AdditionalDirectories)
	if err != nil {
		return fmt.Errorf("failed to get tools: %w", err)
	}

	if *infoTool != "" {
		return displayToolInfo(tools, *infoTool)
	}

	if *searchTerm != "" {
		tools = filterTools(tools, *searchTerm)
	}

	displayTools(tools)
	return nil
}

func loadConfig() (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return config, err
	}
	err = json.Unmarshal(data, &config)
	return config, err
}

func getTools(additionalDirs []string) ([]Tool, error) {
	cachedTools, err := loadCache()
	if err == nil {
		return cachedTools, nil
	}

	var tools []Tool
	var mu sync.Mutex
	var wg sync.WaitGroup

	dirs := append([]string{"./"}, additionalDirs...)

	for _, dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			localTools, err := scanDirectory(d)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning directory %s: %v\n", d, err)
				return
			}
			mu.Lock()
			tools = append(tools, localTools...)
			mu.Unlock()
		}(dir)
	}

	wg.Wait()

	if err := saveCache(tools); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save cache: %v\n", err)
	}

	return tools, nil
}

func scanDirectory(dir string) ([]Tool, error) {
	var tools []Tool

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode().Perm()&0111 != 0 {
			description, _ := getToolDescription(path)
			tools = append(tools, Tool{
				Name:        filepath.Base(path),
				Description: description,
				Path:        path,
			})
		}
		return nil
	})

	return tools, err
}

func getToolDescription(path string) (string, error) {
	cmd := exec.Command(path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			return line, nil
		}
	}
	return "", fmt.Errorf("no description found")
}

func loadCache() ([]Tool, error) {
	var tools []Tool
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &tools)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > cacheDuration {
		return nil, fmt.Errorf("cache expired")
	}

	return tools, nil
}

func saveCache(tools []Tool) error {
	data, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cacheFile, data, 0644)
}

func filterTools(tools []Tool, term string) []Tool {
	var filtered []Tool
	for _, tool := range tools {
		if strings.Contains(strings.ToLower(tool.Name), strings.ToLower(term)) ||
			strings.Contains(strings.ToLower(tool.Description), strings.ToLower(term)) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func displayTools(tools []Tool) {
	for _, tool := range tools {
		fmt.Printf("%s: %s\n", tool.Name, tool.Description)
	}
}

func displayToolInfo(tools []Tool, name string) error {
	for _, tool := range tools {
		if tool.Name == name {
			fmt.Printf("Name: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Printf("Path: %s\n", tool.Path)
			return nil
		}
	}
	return fmt.Errorf("tool not found: %s", name)
}
