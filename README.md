# DevTainr

Setup your VSCode development container configuration files with style 🦗

## Features

Install development container configuration files to your repository for one of the following:

- [`qmcgaw/godevcontainer`](https://github.com/qdm12/godevcontainer)
- [`qmcgaw/reactdevcontainer`](https://github.com/qdm12/reactdevcontainer)
- [`qmcgaw/rustdevcontainer`](https://github.com/qdm12/rustdevcontainer)
- [`qmcgaw/nodedevcontainer`](https://github.com/qdm12/nodedevcontainer)
- [`qmcgaw/latexdevcontainer`](https://github.com/qdm12/latexdevcontainer)
- [`qmcgaw/basedevcontainer`](https://github.com/qdm12/basedevcontainer)

## Usage

### Binary

1. Download the binary for your machine from [the last release page](https://github.com/qdm12/devtainr/releases/latest)
1. If you are on Linux or OSX, make it executable with:

    ```sh
    chmod +x devtainr
    ```

1. Run it with

    ```sh
    ./devtainr -dev go -name projectname
    📁 Creating .devcontainer directory...✔️
    📥 Downloading .dockerignore...✔️
    📥 Downloading Dockerfile...✔️
    📥 Downloading README.md...✔️
    📥 Downloading devcontainer.json...✔️
    📥 Downloading docker-compose.yml...✔️
    ✏️ Setting name to project-dev...✔️
    🦾 Your go development container configuration is ready! 🚀

    # More information:
    ./devtainr -help
    ```

### Docker

```sh
docker run -it --rm --user="$(id -u):$(id -g)" -v "/yourrepopath:/repository" qmcgaw/devtainr -dev go -path /repository -name projectname
📁 Creating .devcontainer directory...✔️
📥 Downloading .dockerignore...✔️
📥 Downloading Dockerfile...✔️
📥 Downloading README.md...✔️
📥 Downloading devcontainer.json...✔️
📥 Downloading docker-compose.yml...✔️
✏️ Setting name to project-dev...✔️
🦾 Your go development container configuration is ready! 🚀

# More information
docker run -it --rm qmcgaw/devtainr -help
```

## Platforms supported

- `linux/amd64`
- `linux/386`
- If you need one more, please [create an issue](https://github.com/qdm12/devtainr/issues/new)

## Build it yourself

Install Go, then either

- Download it on your machine:

  ```sh
  go get github.com/qdm12/devtainr/cmd/devtainr
  ```

- Clone this repository and build it:

  ```sh
  GOARCH=amd64 go build cmd/devtainr/main.go
  ```
