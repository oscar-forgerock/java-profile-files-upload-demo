package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	javaPID     = "1"
	defaultDest = "/tmp/jfr"
)

type Config struct {
	Namespace   string
	PodName     string
	Container   string
	PodSelector string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Parse flags
	config := parseConfig()

	// Resolve pod name if using selector
	if err := resolvePod(&config); err != nil {
		fmt.Printf("Error resolving pod: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "create":
		handleCreate(config)
	case "stop":
		handleStop(config)
	case "list":
		handleList(config)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("JFR CLI - Manage JFR recordings in Kubernetes pods")
	fmt.Println("\nUsage:")
	fmt.Println("  jfr-cli [flags] <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  create [duration] [name]  - Create a new JFR recording")
	fmt.Println("  stop <name>                - Stop a JFR recording by name")
	fmt.Println("  list                       - List all active JFR recordings")
	fmt.Println("\nFlags:")
	fmt.Println("  -n, --namespace <ns>       - Kubernetes namespace (default: default)")
	fmt.Println("  -p, --pod <name>           - Pod name")
	fmt.Println("  -c, --container <name>     - Container name (default: java-app)")
	fmt.Println("  -l, --selector <selector>  - Pod label selector (e.g., app=java-jfr-with-sidecar)")
	fmt.Println("\nExamples:")
	fmt.Println("  jfr-cli -p java-jfr-with-sidecar-0 create 60s")
	fmt.Println("  jfr-cli -l app=java-jfr-with-sidecar create 60s my-profile")
	fmt.Println("  jfr-cli -p java-jfr-with-sidecar-0 stop my-profile")
	fmt.Println("  jfr-cli -p java-jfr-with-sidecar-0 list")
}

func parseConfig() Config {
	config := Config{
		Namespace: "default",
		Container: "java-app",
	}

	// Simple flag parsing
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-n", "--namespace":
			if i+1 < len(os.Args) {
				config.Namespace = os.Args[i+1]
				i++
			}
		case "-p", "--pod":
			if i+1 < len(os.Args) {
				config.PodName = os.Args[i+1]
				i++
			}
		case "-c", "--container":
			if i+1 < len(os.Args) {
				config.Container = os.Args[i+1]
				i++
			}
		case "-l", "--selector":
			if i+1 < len(os.Args) {
				config.PodSelector = os.Args[i+1]
				i++
			}
		}
	}

	return config
}

func resolvePod(config *Config) error {
	if config.PodName != "" {
		return nil
	}

	if config.PodSelector == "" {
		// Default selector
		config.PodSelector = "app=java-jfr-with-sidecar"
	}

	// Get pod by selector
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", config.Namespace,
		"-l", config.PodSelector,
		"-o", "jsonpath={.items[0].metadata.name}")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get pod: %v, output: %s", err, string(output))
	}

	podName := strings.TrimSpace(string(output))
	if podName == "" {
		return fmt.Errorf("no pod found with selector: %s", config.PodSelector)
	}

	config.PodName = podName
	fmt.Printf("Using pod: %s\n", podName)
	return nil
}

func execInPod(config Config, command ...string) (string, error) {
	args := []string{"exec", "-n", config.Namespace, config.PodName, "-c", config.Container, "--"}
	args = append(args, command...)

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func handleCreate(config Config) {
	duration := "60s"
	name := generateRecordingName()

	// Parse remaining args (skip flags)
	args := getNonFlagArgs()
	if len(args) > 1 {
		duration = args[1]
	}
	if len(args) > 2 {
		name = args[2]
	}

	filename := fmt.Sprintf("%s.jfr", name)
	filepath := fmt.Sprintf("%s/%s", defaultDest, filename)

	// Build jcmd command
	jfrSettings := fmt.Sprintf("duration=%s,name=%s,filename=%s", duration, name, filepath)

	fmt.Printf("Creating JFR recording...\n")
	output, err := execInPod(config, "jcmd", javaPID, "JFR.start", jfrSettings)
	if err != nil {
		fmt.Printf("Error starting JFR recording: %v\n", err)
		fmt.Printf("Output: %s\n", output)
		os.Exit(1)
	}

	fmt.Printf("✓ JFR recording started successfully\n")
	fmt.Printf("  Pod:      %s\n", config.PodName)
	fmt.Printf("  Name:     %s\n", name)
	fmt.Printf("  Duration: %s\n", duration)
	fmt.Printf("  Filename: %s\n", filename)
	fmt.Printf("  Path:     %s\n", filepath)
	fmt.Printf("\nOutput:\n%s\n", output)
}

func handleStop(config Config) {
	args := getNonFlagArgs()
	if len(args) < 2 {
		fmt.Println("Error: recording name required")
		fmt.Println("Usage: jfr-cli stop <name>")
		os.Exit(1)
	}

	name := args[1]

	fmt.Printf("Stopping JFR recording...\n")
	output, err := execInPod(config, "jcmd", javaPID, "JFR.stop", fmt.Sprintf("name=%s", name))
	if err != nil {
		fmt.Printf("Error stopping JFR recording: %v\n", err)
		fmt.Printf("Output: %s\n", output)
		os.Exit(1)
	}

	fmt.Printf("✓ JFR recording stopped successfully\n")
	fmt.Printf("  Pod:  %s\n", config.PodName)
	fmt.Printf("  Name: %s\n", name)
	fmt.Printf("\nOutput:\n%s\n", output)
}

func handleList(config Config) {
	fmt.Printf("Listing JFR recordings...\n")
	output, err := execInPod(config, "jcmd", javaPID, "JFR.check")
	if err != nil {
		fmt.Printf("Error listing JFR recordings: %v\n", err)
		fmt.Printf("Output: %s\n", output)
		os.Exit(1)
	}

	// Parse the output to extract recordings
	recordings := parseRecordings(output)

	fmt.Printf("\nPod: %s\n", config.PodName)
	if len(recordings) == 0 {
		fmt.Println("No active JFR recordings found")
	} else {
		fmt.Printf("Active JFR recordings (%d):\n\n", len(recordings))
		for _, rec := range recordings {
			fmt.Printf("  ID:    %s\n", rec.ID)
			fmt.Printf("  Name:  %s\n", rec.Name)
			fmt.Printf("  State: %s\n", rec.State)
			fmt.Println()
		}
	}

	fmt.Println("Raw output:")
	fmt.Println(output)
}

type Recording struct {
	ID    string
	Name  string
	State string
}

func parseRecordings(output string) []Recording {
	var recordings []Recording

	// Pattern: Recording <id>: name=<name> (state)
	re := regexp.MustCompile(`Recording (\d+):\s+name=([^\s]+)\s+\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) == 4 {
			recordings = append(recordings, Recording{
				ID:    match[1],
				Name:  match[2],
				State: match[3],
			})
		}
	}

	return recordings
}

func generateRecordingName() string {
	now := time.Now()
	// Format: jfr_2026-01-22T08-30-15+00-00
	timestamp := now.Format(time.RFC3339)
	// Replace colons with hyphens for filesystem compatibility
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	return fmt.Sprintf("jfr_%s", timestamp)
}

func getNonFlagArgs() []string {
	var result []string
	skip := false

	for i, arg := range os.Args {
		if i == 0 {
			continue // skip program name
		}

		if skip {
			skip = false
			continue
		}

		if strings.HasPrefix(arg, "-") {
			// This is a flag, skip it and potentially its value
			if arg == "-n" || arg == "--namespace" ||
				arg == "-p" || arg == "--pod" ||
				arg == "-c" || arg == "--container" ||
				arg == "-l" || arg == "--selector" {
				skip = true
			}
			continue
		}

		result = append(result, arg)
	}

	return result
}
