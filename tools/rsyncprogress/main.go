package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var (
	verbose     bool
	customChar  string
	customColor string
	refreshRate int
	width       int
	colorMap    = map[string]termbox.Attribute{
		"black":   termbox.ColorBlack,
		"red":     termbox.ColorRed,
		"green":   termbox.ColorGreen,
		"yellow":  termbox.ColorYellow,
		"blue":    termbox.ColorBlue,
		"magenta": termbox.ColorMagenta,
		"cyan":    termbox.ColorCyan,
		"white":   termbox.ColorWhite,
	}
)

type Progress struct {
	CurrentFile     string
	OverallProgress float64
	TransferSpeed   float64
	RemainingTime   time.Duration
	TotalFiles      int
	ProcessedFiles  int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.StringVar(&customChar, "char", "█", "Custom character for progress bar")
	flag.StringVar(&customColor, "color", "cyan", "Color of the progress bar (black, red, green, yellow, blue, magenta, cyan, white)")
	flag.IntVar(&refreshRate, "refresh", 100, "Refresh rate in milliseconds")
	flag.IntVar(&width, "width", 50, "Width of the progress bar")
	flag.Parse()

	if err := termbox.Init(); err != nil {
		return fmt.Errorf("failed to initialize termbox: %w", err)
	}
	defer termbox.Close()

	progress := &Progress{}
	errChan := make(chan error)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		termbox.Close()
		fmt.Println("Program terminated by user")
		os.Exit(0)
	}()

	go readInput(progress, errChan)
	go updateDisplay(progress, errChan)

	return <-errChan
}

func readInput(progress *Progress, errChan chan<- error) {
	reader := bufio.NewReader(os.Stdin)
	fileRegex := regexp.MustCompile(`^(.+)$`)
	progressRegex := regexp.MustCompile(`^\s*(\d+)\s+(\d+)%\s+(\d+\.\d+\w+/s)\s+(\d+:\d+:\d+)`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				errChan <- nil
				return
			}
			errChan <- fmt.Errorf("error reading input: %w", err)
			return
		}

		line = strings.TrimSpace(line)

		if fileMatch := fileRegex.FindStringSubmatch(line); len(fileMatch) > 1 {
			progress.CurrentFile = fileMatch[1]
			progress.ProcessedFiles++
		} else if progressMatch := progressRegex.FindStringSubmatch(line); len(progressMatch) > 4 {
			progress.TotalFiles, _ = strconv.Atoi(progressMatch[1])
			overallProgress, _ := strconv.ParseFloat(progressMatch[2], 64)
			progress.OverallProgress = overallProgress / 100
			progress.TransferSpeed, _ = parseTransferSpeed(progressMatch[3])
			progress.RemainingTime, _ = time.ParseDuration(progressMatch[4])
		}

		if verbose {
			fmt.Println(line)
		}
	}
}

func updateDisplay(progress *Progress, errChan chan<- error) {
	ticker := time.NewTicker(time.Duration(refreshRate) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if err := render(progress); err != nil {
			errChan <- fmt.Errorf("error rendering display: %w", err)
			return
		}
	}
}

func render(progress *Progress) error {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	renderProgressBar(progress)
	renderStats(progress)

	return termbox.Flush()
}

func renderProgressBar(progress *Progress) {
	barColor := colorMap[customColor]
	if barColor == 0 {
		barColor = termbox.ColorCyan
	}

	filled := int(float64(width) * progress.OverallProgress)
	for i := 0; i < width; i++ {
		x := i + 2
		y := 2
		if i < filled {
			drawString(x, y, customChar, barColor, termbox.ColorDefault)
		} else {
			drawString(x, y, "-", termbox.ColorWhite, termbox.ColorDefault)
		}
	}
}

func renderStats(progress *Progress) {
	drawString(2, 4, fmt.Sprintf("Current File: %s", progress.CurrentFile), termbox.ColorWhite, termbox.ColorDefault)
	drawString(2, 5, fmt.Sprintf("Overall Progress: %.2f%%", progress.OverallProgress*100), termbox.ColorWhite, termbox.ColorDefault)
	drawString(2, 6, fmt.Sprintf("Transfer Speed: %.2f MB/s", progress.TransferSpeed), termbox.ColorWhite, termbox.ColorDefault)
	drawString(2, 7, fmt.Sprintf("Remaining Time: %s", progress.RemainingTime.Round(time.Second)), termbox.ColorWhite, termbox.ColorDefault)
	drawString(2, 8, fmt.Sprintf("Files Processed: %d / %d", progress.ProcessedFiles, progress.TotalFiles), termbox.ColorWhite, termbox.ColorDefault)
}

func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for _, r := range s {
		termbox.SetCell(x, y, r, fg, bg)
		x += runewidth.RuneWidth(r)
	}
}

func parseTransferSpeed(speed string) (float64, error) {
	value, err := strconv.ParseFloat(speed[:len(speed)-4], 64)
	if err != nil {
		return 0, err
	}

	unit := speed[len(speed)-4:]
	switch unit {
	case "KB/s":
		return value / 1024, nil
	case "MB/s":
		return value, nil
	case "GB/s":
		return value * 1024, nil
	default:
		return 0, fmt.Errorf("unknown transfer speed unit: %s", unit)
	}
}
