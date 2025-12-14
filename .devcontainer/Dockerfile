FROM ubuntu:22.04

# Evitar prompts interativos
ENV DEBIAN_FRONTEND=noninteractive

# Instalar dependências básicas
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    zsh \
    sudo \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Criar usuário vscode
RUN useradd -m -s /bin/zsh vscode && \
    echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Instalar Go 1.25
RUN curl -OL https://golang.org/dl/go1.25.4.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz && \
    rm go1.25.4.linux-amd64.tar.gz

# Instalar Neovim 0.11.5
RUN curl -LO https://github.com/neovim/neovim/releases/download/v0.11.5/nvim-linux-x86_64.tar.gz && \
    tar -C /usr/local -xzf nvim-linux-x86_64.tar.gz --strip-components=1 && \
    rm nvim-linux-x86_64.tar.gz

# Instalar GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && apt-get install -y gh

# Configurar environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/vscode/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Mudar para usuário vscode
USER vscode
WORKDIR /home/vscode

# Instalar Oh My Zsh e plugins
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended && \
    git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions && \
    git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting

# Configurar Zsh
RUN echo 'export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"' >> ~/.zshrc && \
    echo 'export GOPATH="$HOME/go"' >> ~/.zshrc && \
    echo 'alias nv="nvim"' >> ~/.zshrc && \
    echo 'alias ll="ls -la"' >> ~/.zshrc && \
    sed -i 's/ZSH_THEME=.*/ZSH_THEME="robbyrussell"/' ~/.zshrc && \
    sed -i 's/plugins=(git)/plugins=(git zsh-autosuggestions zsh-syntax-highlighting)/' ~/.zshrc

# Instalar LazyVim
RUN git clone https://github.com/LazyVim/starter ~/.config/nvim --depth=1 && \
    rm -rf ~/.config/nvim/.git

# Instalar ferramentas Go
RUN go install golang.org/x/tools/gopls@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install golang.org/x/tools/cmd/goimports@latest

WORKDIR /workspace
