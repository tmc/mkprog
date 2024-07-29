package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Type        string `json:"type"`
}

type Config struct {
	AdditionalDirectories []string `json:"additionalDirectories"`
}

var (
	cache     map[string][]Tool
	cacheLock sync.RWMutex
	cacheFile = filepath.Join(os.TempDir(), "list-tools-cache.json")
	config    Config
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchTerm := searchCmd.String("term", "", "Search term for filtering tools")

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	toolName := infoCmd.String("tool", "", "Tool name for detailed information")

	if len(os.Args) < 2 {
		return listAllTools()
	}

	switch os.Args[1] {
	case "search":
		searchCmd.Parse(os.Args[2:])
		return searchTools(*searchTerm)
	case "info":
		infoCmd.Parse(os.Args[2:])
		return displayToolInfo(*toolName)
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func listAllTools() error {
	tools, err := getAllTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		fmt.Printf("%s (%s): %s\n", tool.Name, tool.Type, tool.Description)
	}

	return nil
}

func searchTools(term string) error {
	tools, err := getAllTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		if strings.Contains(strings.ToLower(tool.Name), strings.ToLower(term)) ||
			strings.Contains(strings.ToLower(tool.Description), strings.ToLower(term)) {
			fmt.Printf("%s (%s): %s\n", tool.Name, tool.Type, tool.Description)
		}
	}

	return nil
}

func displayToolInfo(name string) error {
	tools, err := getAllTools()
	if err != nil {
		return err
	}

	for _, tool := range tools {
		if tool.Name == name {
			fmt.Printf("Name: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Printf("Location: %s\n", tool.Location)
			fmt.Printf("Type: %s\n", tool.Type)
			return nil
		}
	}

	return fmt.Errorf("tool not found: %s", name)
}

func getAllTools() ([]Tool, error) {
	cacheLock.RLock()
	cachedTools, ok := cache["all"]
	cacheLock.RUnlock()

	if ok {
		return cachedTools, nil
	}

	var tools []Tool
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(3)

	go func() {
		defer wg.Done()
		systemTools, err := getSystemTools()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting system tools: %v\n", err)
			return
		}
		mu.Lock()
		tools = append(tools, systemTools...)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		customTools, err := getCustomTools()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting custom tools: %v\n", err)
			return
		}
		mu.Lock()
		tools = append(tools, customTools...)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		toolchainTools, err := getToolchainTools()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting toolchain tools: %v\n", err)
			return
		}
		mu.Lock()
		tools = append(tools, toolchainTools...)
		mu.Unlock()
	}()

	wg.Wait()

	cacheLock.Lock()
	cache["all"] = tools
	cacheLock.Unlock()

	go saveCache()

	return tools, nil
}

func getSystemTools() ([]Tool, error) {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	var tools []Tool
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			files, err := ioutil.ReadDir(p)
			if err != nil {
				return
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				if runtime.GOOS == "darwin" {
					if !isSignedBinary(filepath.Join(p, file.Name())) {
						continue
					}
				}

				tool := Tool{
					Name:        file.Name(),
					Description: "System tool",
					Location:    filepath.Join(p, file.Name()),
					Type:        "System",
				}

				mu.Lock()
				tools = append(tools, tool)
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()
	return tools, nil
}

func getCustomTools() ([]Tool, error) {
	customDir := filepath.Join(os.Getenv("HOME"), ".toolchain")
	return getToolsFromDir(customDir, "Custom")
}

func getToolchainTools() ([]Tool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return getToolsFromDir(currentDir, "Toolchain")
}

func getToolsFromDir(dir string, toolType string) ([]Tool, error) {
	var tools []Tool
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode().Perm()&0111 != 0 {
			tool := Tool{
				Name:        info.Name(),
				Description: fmt.Sprintf("%s tool", toolType),
				Location:    path,
				Type:        toolType,
			}
			tools = append(tools, tool)
		}
		return nil
	})
	return tools, err
}

func isSignedBinary(path string) bool {
	cmd := exec.Command("codesign", "-v", path)
	return cmd.Run() == nil
}

func loadCache() {
	cache = make(map[string][]Tool)
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &cache)
}

func saveCache() {
	cacheLock.RLock()
	defer cacheLock.RUnlock()

	data, err := json.Marshal(cache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling cache: %v\n", err)
		return
	}

	err = ioutil.WriteFile(cacheFile, data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving cache: %v\n", err)
	}
}

func loadConfig() error {
	configFile := filepath.Join(os.Getenv("HOME"), ".list-tools.json")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &config)
}

func init() {
	loadCache()
	if err := loadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
	}
}
