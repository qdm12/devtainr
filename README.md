# DevTainr

Setup your VSCode development container configuration files with style 🦗

## Features

Install development container configuration files to your repository for one of the following:

- [`qmcgaw/godevcontainer`](https://github.com/qdm12/godevcontainer)
- [`qmcgaw/reactdevcontainer`](https://github.com/qdm12/reactdevcontainer)
- [`qmcgaw/rustdevcontainer`](https://github.com/qdm12/rustdevcontainer)
- [`qmcgaw/nodedevcontainer`](https://github.com/qdm12/nodedevcontainer)

## Usage

### Binary

```sh
VERSION=v0.1.0
ARCH=amd64

wget -O devtainr "https://github.com/qdm12/devtainr/releases/download/$VERSION/xcputranslate_$VERSION_linux_$ARCH"
chmod 500 devtainr

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
docker run -it --rm -v "/yourrepopath:/repository" qmcgaw/devtainr:v0.1.0 -dev go -path /repository -name projectname
📁 Creating .devcontainer directory...✔️
📥 Downloading .dockerignore...✔️
📥 Downloading Dockerfile...✔️
📥 Downloading README.md...✔️
📥 Downloading devcontainer.json...✔️
📥 Downloading docker-compose.yml...✔️
✏️ Setting name to project-dev...✔️
🦾 Your go development container configuration is ready! 🚀

# More information
docker run -it --rm qmcgaw/devtainr:v0.1.0 -help
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
