# GoDeGraph 📊

An interactive visualization tool for exploring Go package dependencies. GoDeGraph generates an HTML-based interactive graph that helps you understand the relationships between packages in your Go projects.

## ✨ Features

- 🎯 Interactive D3.js-based visualization
- 🔍 Zoomable and pannable interface
- 🎨 Module-based color coding
- 🔗 Dependency link visualization
- ⚡️ Multi-node selection support
- 🔄 Cross-module dependency filtering
- 💡 Detailed tooltips with import information

## 📦 Installation

```bash
go install github.com/philous/godegraph/cmd/godegraph@latest
```

Make sure your Go environment is properly set up and `$GOPATH/bin` is in your PATH.

## 🚀 Usage

```bash
godegraph [options] [working_directory]
```

### 🔧 Options

- `-ignore string`: Comma-separated list of paths to ignore (relative to root directory)

### 📝 Arguments

- `working_directory`: The root directory of the Go project (default: current directory)

### 💻 Examples

```bash
# Generate dependency graph for current directory
godegraph

# Ignore specific paths
godegraph -ignore "tests,vendor"

# Generate graph for specific project
godegraph -ignore "scripts,docs" /path/to/project
```

## 🎮 Visualization Features

### 🔵 Node Types
- **Module Root**: Larger circle with border
- **Package**: Medium circle
- **Folder**: Small circle

### 🎨 Color Scheme
- **Green**: All dependencies (default view)
- **Orange**: Outgoing dependencies
- **Blue**: Incoming dependencies
- **Red**: Selected node
- **Gray**: Background/non-selected dependencies

### 🖱️ Interactions
- Click to select/deselect nodes
- Shift+Click for multi-node selection
- Hover for detailed tooltips
- Mouse wheel to zoom
- Background click to reset view

### 🎛️ Toggle Controls
- Show/hide imports
- Show/hide imported-by relationships
- Filter cross-module dependencies

## 📤 Output

The tool generates a `dependency_graph.html` file in the working directory. Open this file in a web browser to explore your project's dependencies interactively.

## ⚙️ Requirements

- Go 1.16 or later
- Modern web browser for visualization

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.