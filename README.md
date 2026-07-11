# gflux

`gflux` is a fast grep-based pattern finder.

---

## Installation Guide

Follow these steps to install Go, configure your environment path, and install `gflux` on your system.

### 1. Install Go

If you do not have Go installed, download and install it from the official site:

- **Download Link**: [go.dev/dl](https://go.dev/dl)
- **Linux/macOS**: Download the archive, extract it to `/usr/local/go` (e.g., `tar -C /usr/local -xzf goX.Y.Z.linux-amd64.tar.gz`).
- **Windows**: Download the MSI installer and run it.

To verify the installation, run:
```bash
go version
```

---

### 2. Configure Environment Variables

You need to ensure that the Go binary directory (`$GOPATH/bin` or `$HOME/go/bin`) is in your system's executable search path (`$PATH`).

#### For macOS / Linux

Select the configuration command based on your default shell:

##### **For Bash (`~/.bashrc` or `~/.bash_profile`)**
Append the Go environment settings:
```bash
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.bashrc
```

##### **For Zsh (`~/.zshrc`)**
Append the Go environment settings:
```bash
echo 'export GOPATH=$HOME/go' >> ~/.zshrc
echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.zshrc
```

##### **For Bourne Shell / generic POSIX (`~/.profile`)**
Append the Go environment settings:
```bash
echo 'export GOPATH=$HOME/go' >> ~/.profile
echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.profile
```

---

### 3. Reload Shell Configuration

To apply the path changes immediately in your current terminal session, source the configuration file:

- **For Bash**:
  ```bash
  source ~/.bashrc
  ```
- **For Zsh**:
  ```bash
  source ~/.zshrc
  ```
- **For Sh/POSIX**:
  ```bash
  source ~/.profile
  ```

---

### 4. Install gflux

Run the following command to download, compile, and install `gflux` directly using the Go toolchain:

```bash
go install -v github.com/hackthacker/gflux@latest
```

---

### 5. Verify Installation

Once installed, check that the binary is available in your path:

```bash
gflux -h
```
