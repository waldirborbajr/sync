gocontainer

A focused Dev Container setup for Go development. The repository provides a Dockerfile under `.devcontainer/` and a `devcontainer.json` that references it.

## Repository structure

```
.devcontainer/
├── Dockerfile           # Container image: Ubuntu + Go + tooling
├── devcontainer.json    # VS Code Dev Container config
```

## What this image provides

- Base: `ubuntu:22.04`
- Go 1.25 (installed to `/usr/local/go`)
- Neovim (v0.11.5)
- GitHub CLI (`gh`)
- Oh My Zsh, zsh-autosuggestions, zsh-syntax-highlighting
- LazyVim starter config
- Common Go tooling installed to user `vscode` (`gopls`, `dlv`, `staticcheck`, `goimports`)
- Environment: `REMOTE_USER` is `vscode`, `GOPATH=/home/vscode/go`, PATH set for Go and GOPATH bin

## Open in VS Code (recommended)

1. Open this folder in VS Code.
2. Command Palette → `Dev Containers: Reopen in Container`.

The included `devcontainer.json` points to `.devcontainer/Dockerfile` and sets the remote user to `vscode`.

## Manual build and run (Docker)

Build the image from the repository root:

```sh
docker build -f .devcontainer/Dockerfile -t gocontainer:latest .
```

Run an interactive shell with your current workspace mounted:

```sh
docker run --rm -it \
	-v "$PWD":/workspace \
	-w /workspace \
	gocontainer:latest zsh
```

Notes:
- Use `zsh` inside the container (the image configures Oh My Zsh for the `vscode` user).
- Adjust mounts/ports as needed for services your project exposes.

## Troubleshooting

- If builds fail due to network or package repository errors, re-run the `docker build` command; temporary APT issues are common.
- To inspect the image, run an interactive shell and check installed tools (`go version`, `nvim --version`, `gh --version`).

## Contributing

Improvements welcome: update the `Dockerfile`, `devcontainer.json`, or this `README.md`. Open a PR with changes and rationale.

seu-projeto/
├── .devcontainer/
│   ├── devcontainer.json
│   ├── setup.sh
│   ├── setup-lazyvim.sh
│   └── setup-zsh.sh
└── (seus arquivos do projeto)
```

```sh
chmod +x .devcontainer/*.sh
```

