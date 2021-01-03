package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/qdm12/devtainr/internal/params"
	"github.com/qdm12/devtainr/internal/setup"
)

//nolint:gochecknoglobals
var (
	version   = "unknown"
	commit    = "unknown"
	buildDate = "an unknown date"
)

func main() {
	ctx := context.Background()
	os.Exit(_main(ctx, os.Args))
}

func _main(ctx context.Context, args []string) int {
	fmt.Printf("🤖 Version %s (commit %s built on %s)\n",
		version, commit, buildDate)

	flagSet := flag.NewFlagSet(args[0], flag.ExitOnError)
	dev := flagSet.String("dev", "go", "can be one of: go, react, rust, node")
	repoPath := flagSet.String("path", ".", "path to the repository")
	namePtr := flagSet.String("name", "project-dev", "name of the development container")
	if err := flagSet.Parse(args[1:]); err != nil {
		fmt.Println("❌")
		fmt.Println(err)
		return 1
	}

	repository, err := params.GetRepository(*dev)
	if err != nil {
		fmt.Println("❌")
		fmt.Println(err)
		return 1
	}

	*repoPath = filepath.Clean(*repoPath)
	devcontainerPath := filepath.Join(*repoPath, ".devcontainer")

	name := *namePtr

	fmt.Print("📁 Creating .devcontainer directory...")
	err = os.Mkdir(devcontainerPath, 0700)
	if err != nil {
		fmt.Println("❌")
		fmt.Println(err)
		return 1
	}
	fmt.Println("✔️")

	const httpTimeout = 5 * time.Second
	client := &http.Client{Timeout: httpTimeout}
	defer client.CloseIdleConnections()

	baseURL := "https://raw.githubusercontent.com/" + repository + "/master/.devcontainer"

	devcontainerFilenames := [...]string{
		".dockerignore",
		"Dockerfile",
		"README.md",
		"devcontainer.json",
		"docker-compose.yml",
	}
	for _, filename := range devcontainerFilenames {
		fmt.Print("📥 Downloading ", filename, "...")
		err := setup.MirrorFile(ctx,
			client, baseURL, devcontainerPath, filename)
		if err != nil {
			fmt.Println("❌")
			fmt.Println(err)
			return 1
		}
		fmt.Println("✔️")
	}

	fmt.Print("✏️ Setting name to ", name, "...")
	jsonFilepath := filepath.Join(devcontainerPath, "devcontainer.json")
	if err := setup.ChangeName(jsonFilepath, name); err != nil {
		fmt.Println("❌")
		fmt.Println(err)
		return 1
	}
	fmt.Println("✔️")

	fmt.Println("🦾 Your " + *dev + " development container configuration is ready! 🚀")

	return 0
}
