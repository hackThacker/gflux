# gflux

`gflux` is a fast grep-based pattern finder.

---

## Installation Guide

Follow these steps to install Go, configure your environment, and install `gflux` on your system.

### 1. Install Go

On Debian/Ubuntu systems, you can install Go using:
```bash
sudo apt install golang-go -y
```

Otherwise, download and install it from the official site:
- **Download Link**: [go.dev/dl](https://go.dev/dl)
- **Linux/macOS Tarball**: Extract to `/usr/local/go`.
- **Windows**: Download and run the MSI installer.

To verify the installation, run:
```bash
go version
```

---

### 2. Configure Environment Variables

Add the Go paths to your shell configuration file.

#### **For Bash (`~/.bashrc`)**
Run these commands to configure the environment:
```bash
echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.bashrc
```

#### **For Zsh (`~/.zshrc`)**
Run these commands to configure the environment:
```bash
echo 'export GOROOT=/usr/local/go' >> ~/.zshrc
echo 'export GOPATH=$HOME/go' >> ~/.zshrc
echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.zshrc
```

---

### 3. Apply/Source Shell Configuration

To apply the path changes immediately in your current terminal session, run:

- **For Bash**:
  ```bash
  source ~/.bashrc
  ```
- **For Zsh**:
  ```bash
  source ~/.zshrc
  ```

---

### 4. Install gflux Binary

Run the following command to download, compile, and install the `gflux` utility:

```bash
go install -v github.com/hackthacker/gflux@latest
```

---

### 5. Install Pattern Files

The Go toolchain's `go install` command only installs the compiled binary. To make `gflux` patterns available, you must copy the `.gf` pattern files to your local configuration folder:

```bash
# Clone the repository
git clone https://github.com/hackthacker/gflux.git
cd gflux

# Copy patterns to the gflux config directory
mkdir -p ~/.gflux
cp -r .gf/* ~/.gflux/
```

---

### 6. Verify Installation

Once installed, check that the binary is available and works correctly:

```bash
gflux -h
```
