package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/qdm12/devtainr/internal/models"
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
	ctx, cancel := context.WithCancel(ctx)
	buildInfo := models.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	}

	errorCh := make(chan error)
	go func() {
		errorCh <- _main(ctx, os.Args, buildInfo)
	}()

	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)

	exitCode := 0
	select {
	case signal := <-signalsCh:
		fmt.Println("\nShutting down: signal", signal)
		exitCode = 1
		cancel()
		timer := time.NewTimer(time.Second)
		select {
		case <-errorCh:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			fmt.Println("Shutdown timed out")
		}
	case err := <-errorCh:
		if err != nil {
			fmt.Println("Fatal error:", err)
			exitCode = 1
		}
		cancel()
	}
	os.Exit(exitCode)
}

func _main(ctx context.Context, args []string, buildInfo models.BuildInfo) error {
	fmt.Printf("🤖 Version %s (commit %s built on %s)\n",
		buildInfo.Version, buildInfo.Commit, buildInfo.BuildDate)

	flagSet := flag.NewFlagSet(args[0], flag.ExitOnError)
	dev := flagSet.String("dev", "go", "can be one of: go, react, rust, node")
	repoPath := flagSet.String("path", ".", "path to the repository")
	namePtr := flagSet.String("name", "project-dev", "name of the development container")
	if err := flagSet.Parse(args[1:]); err != nil {
		return err
	}

	repository, err := params.GetRepository(*dev)
	if err != nil {
		return err
	}

	*repoPath = filepath.Clean(*repoPath)
	devcontainerPath := filepath.Join(*repoPath, ".devcontainer")

	name := *namePtr

	fmt.Print("📁 Creating .devcontainer directory...")
	err = os.Mkdir(devcontainerPath, 0700)
	if err != nil {
		fmt.Println("❌")
		return err
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
			return err
		}
		fmt.Println("✔️")
	}

	fmt.Print("✏️ Setting name to ", name, "...")
	jsonFilepath := filepath.Join(devcontainerPath, "devcontainer.json")
	if err := setup.ChangeName(jsonFilepath, name); err != nil {
		fmt.Println("❌")
		return err
	}
	fmt.Println("✔️")

	fmt.Println("🦾 Your " + *dev + " development container configuration is ready! 🚀")

	return nil
}
