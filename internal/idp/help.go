package idp

import "strings"

const helpText = `
  _   _       _       ____ _     ___
 | | | |_   _| |__   / ___| |   |_ _|
 | |_| | | | | '_ \ | |   | |    | |
 |  _  | |_| | |_) || |___| |___ | |
 |_| |_|\__,_|_.__/  \____|_____|___|

Hub CLI (IDP)

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
  env describe <name>
`

func serviceHelpText() string {
	return strings.TrimSpace(`Usage:
  service list [--namespace|-n <namespace>]
  service describe <service> [--namespace|-n <namespace>]
  service health <service> [--namespace|-n <namespace>]
  service logs <service> [--namespace|-n <namespace>] [--container|-c <container>] [--tail <lines>] [--since <duration>] [--previous]
  service events <service> [--namespace|-n <namespace>] [--tail <count>] [--since <duration>]
  service metrics <service> [--namespace|-n <namespace>] [--window <duration>]
  service traces <service> [--namespace|-n <namespace>] [--hours <hours>] [--limit <count>]
  service ownership <service> [--namespace|-n <namespace>]`) + "\n"
}

func clusterHelpText() string {
	return strings.TrimSpace(`Usage:
  cluster status
  cluster namespaces
  cluster workloads [--namespace|-n <namespace>]`) + "\n"
}

func catalogHelpText() string {
	return strings.TrimSpace(`Usage:
  catalog list [--namespace|-n <namespace>]
  catalog validate [--namespace|-n <namespace>]`) + "\n"
}

func envHelpText() string {
	return strings.TrimSpace(`Usage:
  env list [--namespace|-n <namespace>]
  env describe <name> [--namespace|-n <namespace>] [--kind <ConfigMap|Secret>]`) + "\n"
}
