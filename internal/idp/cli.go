package idp

import (
	"fmt"
	"io"
	"strings"
)

const helpText = `Hub CLI (IDP)

Usage:
  hub-cli <command> [arguments]

Service commands:
  service list
  service describe <service>
  service health <service>
  service logs <service>
  service events <service>
  service metrics <service>
  service traces <service>
  service ownership <service>

Cluster commands:
  cluster status
  cluster namespaces
  cluster workloads

Catalog commands:
  catalog list
  catalog validate

Environment commands:
  env list
  env describe <env>
`

// Run executes the local IDP CLI command dispatcher.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, helpText)
		return 0
	}

	switch args[0] {
	case "service":
		return runService(args[1:], stdout, stderr)
	case "cluster":
		return runCluster(args[1:], stdout, stderr)
	case "catalog":
		return runCatalog(args[1:], stdout, stderr)
	case "env":
		return runEnv(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		fmt.Fprint(stderr, helpText)
		return 1
	}
}

func runService(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, serviceHelpText())
		return 0
	}

	switch args[0] {
	case "list":
		return printCalled(stdout, "idp service list")
	case "describe", "health", "logs", "events", "metrics", "traces", "ownership":
		if len(args) < 2 {
			fmt.Fprintf(stderr, "missing service name for idp service %s\n", args[0])
			return 1
		}
		return printCalled(stdout, "idp service "+args[0]+" for "+args[1])
	default:
		fmt.Fprintf(stderr, "unknown service command: %s\n\n", args[0])
		fmt.Fprint(stderr, serviceHelpText())
		return 1
	}
}

func runCluster(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, clusterHelpText())
		return 0
	}

	switch args[0] {
	case "status", "namespaces", "workloads":
		return printCalled(stdout, "idp cluster "+args[0])
	default:
		fmt.Fprintf(stderr, "unknown cluster command: %s\n\n", args[0])
		fmt.Fprint(stderr, clusterHelpText())
		return 1
	}
}

func runCatalog(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, catalogHelpText())
		return 0
	}

	switch args[0] {
	case "list", "validate":
		return printCalled(stdout, "idp catalog "+args[0])
	default:
		fmt.Fprintf(stderr, "unknown catalog command: %s\n\n", args[0])
		fmt.Fprint(stderr, catalogHelpText())
		return 1
	}
}

func runEnv(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		fmt.Fprint(stdout, envHelpText())
		return 0
	}

	switch args[0] {
	case "list":
		return printCalled(stdout, "idp env list")
	case "describe":
		if len(args) < 2 {
			fmt.Fprint(stderr, "missing environment name for idp env describe\n")
			return 1
		}
		return printCalled(stdout, "idp env describe for "+args[1])
	default:
		fmt.Fprintf(stderr, "unknown env command: %s\n\n", args[0])
		fmt.Fprint(stderr, envHelpText())
		return 1
	}
}

func printCalled(stdout io.Writer, command string) int {
	fmt.Fprintf(stdout, "called %s\n", command)
	return 0
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func serviceHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli service list
  hub-cli service describe <service>
  hub-cli service health <service>
  hub-cli service logs <service>
  hub-cli service events <service>
  hub-cli service metrics <service>
  hub-cli service traces <service>
  hub-cli service ownership <service>`) + "\n"
}

func clusterHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli cluster status
  hub-cli cluster namespaces
  hub-cli cluster workloads`) + "\n"
}

func catalogHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli catalog list
  hub-cli catalog validate`) + "\n"
}

func envHelpText() string {
	return strings.TrimSpace(`Usage:
  hub-cli env list
  hub-cli env describe <env>`) + "\n"
}
