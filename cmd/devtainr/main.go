package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "github.com/breml/rootcerts"
	"github.com/qdm12/devtainr/internal/models"
	"github.com/qdm12/devtainr/internal/params"
	"github.com/qdm12/devtainr/internal/setup"
)

//nolint:gochecknoglobals
var (
	version = "unknown"
	commit  = "unknown"
	created = "an unknown date"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	buildInfo := models.BuildInfo{
		Version: version,
		Commit:  commit,
		Created: created,
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

// devcontainerSubDirectory is the directory holding the development container
// configuration files, both in the source repository and in the destination.
const devcontainerSubDirectory = ".devcontainer"

func _main(ctx context.Context, args []string, buildInfo models.BuildInfo) error {
	fmt.Printf("🤖 Version %s (commit %s built on %s)\n",
		buildInfo.Version, buildInfo.Commit, buildInfo.Created)

	flagSet := flag.NewFlagSet(args[0], flag.ExitOnError)
	dev := flagSet.String("dev", "go", "can be one of: go, rust, latex, base")
	repoPath := flagSet.String("path", ".", "path to the repository")
	namePtr := flagSet.String("name", "project-dev", "name of the development container")
	err := flagSet.Parse(args[1:])
	if err != nil {
		return err
	}

	repository, err := params.GetRepository(*dev)
	if err != nil {
		return err
	}

	*repoPath = filepath.Clean(*repoPath)
	devcontainerPath := filepath.Join(*repoPath, devcontainerSubDirectory)

	const httpTimeout = 30 * time.Second
	client := &http.Client{Timeout: httpTimeout}
	defer client.CloseIdleConnections()

	err = createDirectory(devcontainerPath)
	if err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	err = downloadFiles(ctx, client, repository, devcontainerPath)
	if err != nil {
		return fmt.Errorf("downloading files: %w", err)
	}

	err = setNames(devcontainerPath, *namePtr)
	if err != nil {
		return fmt.Errorf("setting names: %w", err)
	}

	fmt.Println("🦾 Your " + *dev + " development container configuration is ready! 🚀")

	return nil
}

func createDirectory(devcontainerPath string) error {
	fmt.Print("📁 Creating ", devcontainerSubDirectory, " directory...")
	const dirPerms = 0o777
	err := os.MkdirAll(devcontainerPath, dirPerms)
	if err != nil {
		fmt.Println("❌")
		return err
	}
	fmt.Println("✔️")

	return nil
}

// downloadFiles downloads the whole devcontainerSubDirectory directory of the
// repository, sub directories included, and lists what has been written.
func downloadFiles(ctx context.Context, client *http.Client,
	repository, devcontainerPath string,
) error {
	fmt.Print("📥 Downloading ", repository, " "+devcontainerSubDirectory+" directory...")
	entries, err := setup.DownloadFiles(ctx, client, repository,
		devcontainerSubDirectory, devcontainerPath)
	if err != nil {
		fmt.Println("❌")
		return err
	}
	fmt.Println("✔️")

	for _, entry := range entries {
		icon := "📄"
		if entry.IsDir {
			icon = "📁"
		}
		fmt.Println(icon, entry.Path)
	}

	return nil
}

// setNames sets the name of every development container configuration file
// found under devcontainerPath, at any depth. A file which cannot be changed
// does not stop the other ones from being changed.
func setNames(devcontainerPath, name string) error {
	paths, err := setup.DevcontainerFiles(devcontainerPath)
	if err != nil {
		fmt.Println("❌")
		return err
	}

	var errs []error
	for _, path := range paths {
		fmt.Print("✏️ Setting name of ", path, " to ", name, "...")
		err := setup.ChangeName(filepath.Join(devcontainerPath, filepath.FromSlash(path)), name)
		if err != nil {
			fmt.Println("❌")
			errs = append(errs, err)
			continue
		}
		fmt.Println("✔️")
	}

	return errors.Join(errs...)
}
