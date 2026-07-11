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

### 5. Verify Installation & Usage

Once `gflux` is installed, verify that the binary is available and lists the embedded patterns correctly:

```bash
gflux -list
```

---

## Embedded Patterns & Customization

`gflux` is fully self-contained. The default patterns are embedded directly inside the binary. You do not need to download or copy anything to start using it immediately!

### Pattern Resolution Priority
`gflux` looks up patterns using the following priority order:
1. **Local project patterns** (highest priority): `./.gflux/`
2. **User custom patterns**: `~/.gflux/` (also looks in `~/.config/gflux/`, `~/.gf/`, and `~/.config/gf/`)
3. **Embedded default patterns** (lowest priority): Pre-compiled inside the binary.

If a pattern name matches in multiple locations, the higher priority source overrides the lower priority one.

### Grouped Pattern Listing
To see exactly which patterns are available and where they are loaded from, run:
```bash
gflux --list-patterns
```

### Initializing User Custom Patterns
If you want to edit or customize the default patterns, you can initialize the user custom folder and copy the embedded default patterns to it:
```bash
# Initialize and copy patterns (will prompt)
gflux init

# Initialize and copy automatically (force/yes flag)
gflux init -y
```
