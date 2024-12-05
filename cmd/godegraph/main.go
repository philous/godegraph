package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// Add paths to ignore relative to root directory
	ignoredPaths []string
)

// shouldIgnorePath checks if the given path should be ignored
func shouldIgnorePath(path string, rootDir string) bool {
	// Get relative path from root directory
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return false
	}

	// Check if path matches any ignored path
	for _, ignorePath := range ignoredPaths {
		// Convert both paths to use forward slashes for consistent comparison
		ignorePath = filepath.ToSlash(ignorePath)
		relPath = filepath.ToSlash(relPath)

		if strings.HasPrefix(relPath, ignorePath) {
			return true
		}
	}
	return false
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Dependency Graph</title>
    <script src="https://d3js.org/d3.v6.min.js"></script>
    <style>
        body {
            margin: 0;
            font-family: Arial, sans-serif;
            background-color: #f8f9fa;
        }
        #controls {
            position: fixed;
            top: 10px;
            left: 10px;
            z-index: 1000;
            background-color: rgba(255, 255, 255, 0.9);
            padding: 10px;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .toggle-btn {
            padding: 5px 10px;
            margin: 0 5px;
            border: 1px solid #ccc;
            border-radius: 3px;
            background-color: #fff;
            cursor: pointer;
            transition: background-color 0.3s;
        }
        .toggle-btn.active {
            background-color: #007bff;
            color: white;
            border-color: #0056b3;
        }
        .node circle {
            stroke-width: 2px;
            transition: fill 0.3s, stroke 0.3s;
        }
        .node.module-root circle {
            stroke-width: 3px;
            r: 8;
        }
        .node.folder circle {
            fill: inherit;
            stroke: inherit;
        }
        .module-indicator {
            font-size: 10px;
            fill: #fff;
            font-weight: bold;
            text-shadow: 1px 1px 1px rgba(0,0,0,0.5);
        }
        .node.selected circle {
            stroke: #ff0000;  /* red for selected node */
            stroke-width: 3px;
        }
        .node.importing circle {
            stroke: #ff7f0e;  /* orange for nodes that the selected node imports */
            stroke-width: 3px;
            filter: brightness(1.2);
        }
        .node.imported circle {
            stroke: #1f77b4;  /* blue for nodes that import the selected node */
            stroke-width: 3px;
            filter: brightness(1.2);
        }
        .node text {
            font-size: 10px;
            fill: #666;
        }
        .dependency-link {
            fill: none;
            stroke: #999;
            stroke-opacity: 0.3;
            stroke-width: 1.5px;
        }
        .dependency-link.outgoing {
            stroke: #ff7f0e;
            stroke-opacity: 0.6;
        }
        .dependency-link.incoming {
            stroke: #1f77b4;
            stroke-opacity: 0.6;
        }
        .dependency-link.all {
            stroke: #27ae60;  /* darker green color */
            stroke-width: 1.5px;
            stroke-opacity: 0.4;
            fill: none;
        }
        .dependency-link.background {
            stroke: #ddd;
            stroke-width: 1.5px;
            stroke-opacity: 0.1;
            fill: none;
        }
        .tooltip {
            position: absolute;
            padding: 12px;
            background: white;
            border-radius: 5px;
            font-size: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            max-width: 500px;
        }
        .tooltip-title {
            font-weight: bold;
            margin-bottom: 8px;
        }
        .tooltip-module {
            color: #666;
            margin-bottom: 8px;
        }
        .tooltip-section {
            margin-top: 8px;
        }
        .tooltip-list {
            margin: 4px 0;
            padding-left: 20px;
        }
        /* Add legend styles */
        .legend {
            position: fixed;
            top: 20px;
            right: 20px;
            background: rgba(255, 255, 255, 0.9);
            padding: 15px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            font-size: 14px;
        }
        .legend-title {
            font-weight: bold;
            margin-bottom: 10px;
        }
        .legend-item {
            display: flex;
            align-items: center;
            margin: 5px 0;
        }
        .legend-line {
            width: 30px;
            height: 2px;
            margin-right: 8px;
        }
        .legend-circle {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
            border: 2px solid;
        }
        .legend-text {
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div id="controls">
        <button id="toggleOutgoing" class="toggle-btn active">Imports</button>
        <button id="toggleIncoming" class="toggle-btn active">Imported by</button>
        <button id="toggleCrossModule" class="toggle-btn">Cross-module only</button>
    </div>
    <div class="legend">
        <div class="legend-title">Legend</div>
        <div class="legend-item">
            <div class="legend-line" style="background: #27ae60; opacity: 0.4;"></div>
            <span class="legend-text">All Dependencies</span>
        </div>
        <div class="legend-item">
            <div class="legend-line" style="background: #ff7f0e; opacity: 0.4;"></div>
            <span class="legend-text">Outgoing Dependencies</span>
        </div>
        <div class="legend-item">
            <div class="legend-line" style="background: #1f77b4; opacity: 0.4;"></div>
            <span class="legend-text">Incoming Dependencies</span>
        </div>
        <div class="legend-item">
            <div class="legend-circle" style="border-color: #ff0000;"></div>
            <span class="legend-text">Selected Node</span>
        </div>
        <div class="legend-item">
            <div class="legend-circle" style="border-color: #ff7f0e;"></div>
            <span class="legend-text">Imported Node</span>
        </div>
        <div class="legend-item">
            <div class="legend-circle" style="border-color: #1f77b4;"></div>
            <span class="legend-text">Importing Node</span>
        </div>
    </div>
    <div id="tooltip" class="tooltip" style="display: none;"></div>
    <script>
        const data = {{.}};
        let showIncoming = true;
        let showOutgoing = true;
        let showCrossModuleOnly = false;
        let selectedNodeIds = new Set();

        function createHierarchy(data) {
            // Create nodes map first
            const nodesMap = new Map();
            data.nodes.forEach(node => {
                nodesMap.set(node.id, {
                    id: node.id,
                    name: node.id.split("/").pop(),
                    module: node.module,
                    isPackage: true,
                    children: []
                });
            });

            // Create folder nodes and build hierarchy
            const root = {
                id: "",
                name: "root",
                children: []
            };

            data.nodes.forEach(node => {
                const parts = node.id.split("/");
                let currentPath = "";
                let parent = root;

                parts.forEach((part, index) => {
                    if (!part) return;
                    
                    currentPath = currentPath ? currentPath + "/" + part : part;
                    
                    let currentNode;
                    if (index === parts.length - 1) {
                        // This is a package node
                        currentNode = nodesMap.get(currentPath);
                    } else {
                        // This is a folder node
                        if (!nodesMap.has(currentPath)) {
                            currentNode = {
                                id: currentPath,
                                name: part,
                                module: node.module, // Inherit module from the package
                                isPackage: false,
                                children: []
                            };
                            nodesMap.set(currentPath, currentNode);
                        } else {
                            currentNode = nodesMap.get(currentPath);
                        }
                    }

                    // Add to parent if not already there
                    if (!parent.children.find(child => child.id === currentNode.id)) {
                        parent.children.push(currentNode);
                    }
                    parent = currentNode;
                });
            });

            // Add imports information
            data.links.forEach(link => {
                const source = nodesMap.get(link.source);
                const target = nodesMap.get(link.target);
                if (source && target) {
                    if (!source.imports) source.imports = [];
                    if (!target.importedBy) target.importedBy = [];
                    source.imports.push(target.id);
                    target.importedBy.push(source.id);
                }
            });

            return root;
        }

        // Create the hierarchy
        const hierarchyData = createHierarchy(data);
        const root = d3.hierarchy(hierarchyData);

        // Set up the tree layout
        const width = Math.max(window.innerWidth, 1200);  // Minimum width
        const height = Math.max(window.innerHeight * 3, 2000);  // Ensure enough vertical space

        const svg = d3.select("body").append("svg")
            .attr("width", width)
            .attr("height", height)
            .on("click", function(event) {
                if (event.target === this) {
                    selectedNodeIds.clear();
                    updateNodeStyles();
                    updateDependencyVisibility();
                }
            });

        const g = svg.append("g")
            .attr("transform", "translate(40,0)"); // Add some left margin

        const treeLayout = d3.tree()
            .size([height - 100, width - 160])  // Leave space for labels
            .separation(function(a, b) {
                return (a.parent == b.parent ? 4 : 6) * (a.depth === b.depth ? 1.5 : 2);
            });

        treeLayout(root);

        // Initialize module colors
        const moduleColors = new Map();
        data.modules.forEach(module => {
            moduleColors.set(module.modulePath, module.color);
        });

        const tooltip = d3.select("#tooltip");
        const linksGroup = g.append("g").attr("class", "links");
        const dependencyLinksGroup = g.append("g").attr("class", "dependency-links");
        const incomingLinksGroup = g.append("g").attr("class", "incoming-links");
        const nodesGroup = g.append("g").attr("class", "nodes");

        // Create the links
        const links = root.links();
        const link = linksGroup.selectAll(".link")
            .data(links)
            .enter()
            .append("path")
            .attr("class", "link")
            .attr("d", function(d) {
                return "M" + d.source.y + "," + d.source.x +
                       "C" + (d.source.y + d.target.y) / 2 + "," + d.source.x +
                       " " + (d.source.y + d.target.y) / 2 + "," + d.target.x +
                       " " + d.target.y + "," + d.target.x;
            })
            .attr("fill", "none")
            .attr("stroke", "#ccc");

        // Create nodes
        const node = nodesGroup.selectAll(".node")
            .data(root.descendants())
            .enter()
            .append("g")
            .attr("class", function(d) {
                let classes = ["node"];
                if (d.data.isPackage) {
                    classes.push("package");
                } else if (d.data.id === d.data.module) {
                    classes.push("module-root");
                } else {
                    classes.push("folder");
                }
                return classes.join(" ");
            })
            .attr("transform", function(d) {
                return "translate(" + d.y + "," + d.x + ")";
            })
            .on("click", handleNodeClick)
            .on("mouseover", handleNodeMouseOver)
            .on("mouseout", handleNodeMouseOut);

        node.append("circle")
            .attr("r", function(d) {
                if (d.data.id === d.data.module) return 8;
                if (d.data.isPackage) return 6;
                return 4;
            })
            .attr("fill", function(d) {
                if (!d.data || !d.data.module) return "#f8f9fa";
                
                const color = moduleColors.get(d.data.module);
                if (!color) return "#f8f9fa";
                
                // All nodes get their module's color
                return color;
            })
            .attr("stroke", function(d) {
                if (!d.data || !d.data.module) return "#dee2e6";
                
                const color = moduleColors.get(d.data.module);
                if (!color) return "#dee2e6";
                
                return d3.color(color).darker(0.8);
            });

        // Add module indicator for module roots
        node.filter(d => d.data.id === d.data.module)
            .append("text")
            .attr("class", "module-indicator")
            .attr("dy", "-1em")
            .attr("text-anchor", "middle")
            .text("M");

        // Add labels
        node.append("text")
            .attr("dy", "0.35em") // This centers text vertically
            .attr("x", function(d) {
                const radius = d.data.id === d.data.module ? 8 : (d.data.isPackage ? 6 : 4);
                return d.children || d._children ? -radius - 5 : radius + 5;
            })
            .style("text-anchor", function(d) {
                return d.children || d._children ? "end" : "start";
            })
            .text(function(d) { return d.data.name; });

        function handleNodeClick(event, d) {
            if (!event.shiftKey) {
                selectedNodeIds.clear();
            }
            
            if (selectedNodeIds.has(d.data.id)) {
                selectedNodeIds.delete(d.data.id);
            } else {
                selectedNodeIds.add(d.data.id);
            }
            
            updateNodeStyles();
            updateDependencyVisibility();
        }

        function handleNodeMouseOver(event, d) {
            const tooltip = d3.select("#tooltip");
            const [x, y] = d3.pointer(event, document.body);
            
            let content = '<div class="tooltip-title">' + d.data.id + '</div>';
            content += '<div class="tooltip-module">Module: ' + d.data.module + '</div>';
            
            if (d.data.imports && d.data.imports.length > 0) {
                content += '<div class="tooltip-section">Imports (' + d.data.imports.length + '):</div>';
                content += '<ul class="tooltip-list">';
                d.data.imports.forEach(imp => {
                    content += '<li>' + imp + '</li>';
                });
                content += '</ul>';
            } else {
                content += '<div class="tooltip-section">Imports: 0</div>';
            }
            
            if (d.data.importedBy && d.data.importedBy.length > 0) {
                content += '<div class="tooltip-section">Imported by (' + d.data.importedBy.length + '):</div>';
                content += '<ul class="tooltip-list">';
                d.data.importedBy.forEach(imp => {
                    content += '<li>' + imp + '</li>';
                });
                content += '</ul>';
            } else {
                content += '<div class="tooltip-section">Imported by: 0</div>';
            }
            
            tooltip.html(content)
                .style("left", (x + 10) + "px")
                .style("top", (y + 10) + "px")
                .style("display", "block");
        }

        function handleNodeMouseOut() {
            tooltip.style("display", "none");
        }

        function updateNodeStyles() {
            const nodes = nodesGroup.selectAll(".node");
            
            if (selectedNodeIds.size === 0) {
                nodes.classed("selected", false)
                     .classed("imported", false)
                     .classed("importing", false);
                return;
            }

            // Get all imports and importedBy for selected nodes
            let allImports = new Set();
            let allImportedBy = new Set();
            
            selectedNodeIds.forEach(nodeId => {
                const selectedNode = root.descendants().find(d => d.data.id === nodeId);
                if (selectedNode && selectedNode.data) {
                    (selectedNode.data.imports || []).forEach(id => allImports.add(id));
                    (selectedNode.data.importedBy || []).forEach(id => allImportedBy.add(id));
                }
            });

            nodes.classed("selected", d => selectedNodeIds.has(d.data.id))
                 .classed("importing", d => allImports.has(d.data.id) && !selectedNodeIds.has(d.data.id))
                 .classed("imported", d => allImportedBy.has(d.data.id) && !selectedNodeIds.has(d.data.id));
        }

        function generateLinkPath(source, target) {
            const dx = target.x - source.x;
            const dy = target.y - source.y;
            const dr = Math.sqrt(dx * dx + dy * dy);
            return "M" + source.y + "," + source.x + 
                   "C" + (source.y + target.y) / 2 + "," + source.x +
                   " " + (source.y + target.y) / 2 + "," + target.x +
                   " " + target.y + "," + target.x;
        }

        function getModulePath(node) {
            return node.data.module || '';
        }

        function isCrossModuleDependency(sourceNode, targetNode) {
            return getModulePath(sourceNode) !== getModulePath(targetNode);
        }

        function updateDependencyVisibility() {
            // Remove existing dependency links
            dependencyLinksGroup.selectAll(".dependency-link").remove();
            incomingLinksGroup.selectAll(".dependency-link").remove();
            
            const allNodes = root.descendants();
            
            if (selectedNodeIds.size === 0) {
                // When no node is selected, show all dependencies in green
                allNodes.forEach(sourceNode => {
                    if (sourceNode.data.imports) {
                        sourceNode.data.imports.forEach(targetId => {
                            const targetNode = allNodes.find(d => d.data.id === targetId);
                            if (targetNode) {
                                // Skip if we're showing only cross-module dependencies and this is within the same module
                                if (showCrossModuleOnly && !isCrossModuleDependency(sourceNode, targetNode)) {
                                    return;
                                }
                                dependencyLinksGroup.append("path")
                                    .attr("class", "dependency-link all")
                                    .attr("d", generateLinkPath(sourceNode, targetNode))
                                    .style("stroke", "#27ae60")  /* darker green color */
                                    .style("opacity", showOutgoing ? 0.4 : 0)
                                    .style("fill", "none")
                                    .style("stroke-width", "1.5px");
                            }
                        });
                    }
                });
                return;
            }

            // Show dependencies for selected nodes
            selectedNodeIds.forEach(nodeId => {
                const selectedNode = allNodes.find(d => d.data.id === nodeId);
                if (!selectedNode || !selectedNode.data) return;

                // Outgoing dependencies (imports)
                if (showOutgoing && selectedNode.data.imports) {
                    selectedNode.data.imports.forEach(targetId => {
                        const targetNode = allNodes.find(d => d.data.id === targetId);
                        if (targetNode) {
                            if (showCrossModuleOnly && !isCrossModuleDependency(selectedNode, targetNode)) {
                                return;
                            }
                            dependencyLinksGroup.append("path")
                                .attr("class", "dependency-link outgoing")
                                .attr("d", generateLinkPath(selectedNode, targetNode))
                                .style("stroke", "#ff7f0e")
                                .style("opacity", 0.4)
                                .style("fill", "none")
                                .style("stroke-width", "1.5px");
                        }
                    });
                }

                // Incoming dependencies (imported by)
                if (showIncoming && selectedNode.data.importedBy) {
                    selectedNode.data.importedBy.forEach(sourceId => {
                        const sourceNode = allNodes.find(d => d.data.id === sourceId);
                        if (sourceNode) {
                            if (showCrossModuleOnly && !isCrossModuleDependency(sourceNode, selectedNode)) {
                                return;
                            }
                            incomingLinksGroup.append("path")
                                .attr("class", "dependency-link incoming")
                                .attr("d", generateLinkPath(sourceNode, selectedNode))
                                .style("stroke", "#1f77b4")
                                .style("opacity", 0.4)
                                .style("fill", "none")
                                .style("stroke-width", "1.5px");
                        }
                    });
                }
            });

            // Show dimmed dependencies for non-selected nodes
            allNodes.forEach(sourceNode => {
                if (!selectedNodeIds.has(sourceNode.data.id) && sourceNode.data.imports) {
                    sourceNode.data.imports.forEach(targetId => {
                        const targetNode = allNodes.find(d => d.data.id === targetId);
                        if (targetNode) {
                            dependencyLinksGroup.append("path")
                                .attr("class", "dependency-link background")
                                .attr("d", generateLinkPath(sourceNode, targetNode))
                                .style("stroke", "#ddd")
                                .style("opacity", 0.1)
                                .style("fill", "none")
                                .style("stroke-width", "1.5px");
                        }
                    });
                }
            });
        }

        // Initialize dependency visibility to show all dependencies
        updateDependencyVisibility();
        
        // Add event listeners for toggle buttons
        document.getElementById("toggleOutgoing").onclick = function() {
            this.classList.toggle("active");
            showOutgoing = !showOutgoing;
            updateDependencyVisibility();
        };

        document.getElementById("toggleIncoming").onclick = function() {
            this.classList.toggle("active");
            showIncoming = !showIncoming;
            updateDependencyVisibility();
        };

        d3.select("#toggleCrossModule")
            .on("click", function() {
                const btn = d3.select(this);
                showCrossModuleOnly = !showCrossModuleOnly;
                btn.classed("active", showCrossModuleOnly);
                updateDependencyVisibility();
            });

        // Add zoom behavior
        const zoom = d3.zoom()
            .scaleExtent([0.1, 3])
            .on("zoom", (event) => {
                g.attr("transform", event.transform);
            });

        svg.call(zoom);
        
        // Initial zoom to fit the content
        const bounds = g.node().getBBox();
        const fullWidth = bounds.width;
        const fullHeight = bounds.height;
        const scale = Math.min(width / fullWidth, height / fullHeight) * 0.9;
        const translateX = (width - fullWidth * scale) / 2;
        const translateY = (height - fullHeight * scale) / 2;
        
        svg.call(zoom.transform, d3.zoomIdentity
            .translate(translateX, translateY)
            .scale(scale));
    </script>
</body>
</html>
`

type ModuleInfo struct {
	Path       string `json:"path"`       // Relative path to module directory
	Dir        string `json:"dir"`        // Full path to module directory
	Name       string `json:"name"`       // Module name from go.mod
	Color      string `json:"color"`      // Color assigned to module
	ModulePath string `json:"modulePath"` // Full module path from go.mod file
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Graph struct {
	Nodes          []Node                  `json:"nodes"`
	Links          []Link                  `json:"links"`
	SavedPositions map[string]NodePosition `json:"savedPositions,omitempty"`
	Modules        []ModuleInfo            `json:"modules"`
}

type Node struct {
	ID     string `json:"id"`
	Module string `json:"module"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Package struct {
	ImportPath string
	Imports    []string
}

func findModules(rootDir string) ([]ModuleInfo, error) {
	var modules []ModuleInfo
	colors := []string{
		"#3498db", // Blue
		"#9b59b6", // Purple
		"#f1c40f", // Yellow
		"#e67e22", // Orange
		"#1abc9c", // Turquoise
		"#34495e", // Dark Blue
	}
	colorIndex := 0

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if path should be ignored
		if shouldIgnorePath(path, rootDir) {
			return filepath.SkipDir
		}

		if info.Name() == "go.mod" {
			relPath, err := filepath.Rel(rootDir, filepath.Dir(path))
			if err != nil {
				return err
			}

			// Read go.mod file to get module name
			modContent, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// Extract module name from go.mod content
			moduleName := ""
			lines := strings.Split(string(modContent), "\n")
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "module ") {
					moduleName = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "module "))
					break
				}
			}

			if moduleName == "" {
				moduleName = filepath.Base(relPath) // fallback to directory name
			}

			color := colors[colorIndex%len(colors)]
			colorIndex++

			modules = append(modules, ModuleInfo{
				Path:       relPath,
				Dir:        filepath.Dir(path),
				Name:       filepath.Base(relPath),
				Color:      color,
				ModulePath: moduleName,
			})
		}
		return nil
	})

	return modules, err
}

func loadSavedPositions() (map[string]NodePosition, error) {
	data, err := ioutil.ReadFile("node_positions.json")
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]NodePosition), nil
		}
		return nil, err
	}

	var positions map[string]NodePosition
	if err := json.Unmarshal(data, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

func extractPackageDependencies() (*Graph, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}
	fmt.Printf("Current directory: %s\n", currentDir)

	// Find all modules
	modules, err := findModules(currentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find modules: %v", err)
	}

	var allPackages []Package
	for _, module := range modules {
		fmt.Printf("\nProcessing module in directory: %s\n", module.Dir)

		// Change to module directory
		if err := os.Chdir(module.Dir); err != nil {
			fmt.Printf("Warning: failed to change to directory %s: %v\n", module.Dir, err)
			continue
		}

		cmd := exec.Command("go", "list", "-json", "./...")
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Warning: failed to list packages in %s: %v\n", module.Dir, err)
			continue
		}

		decoder := json.NewDecoder(bytes.NewReader(output))
		for decoder.More() {
			var pkg Package
			if err := decoder.Decode(&pkg); err != nil {
				fmt.Printf("Warning: failed to decode package: %v\n", err)
				continue
			}
			allPackages = append(allPackages, pkg)
		}

		// Change back to root directory
		if err := os.Chdir(currentDir); err != nil {
			return nil, fmt.Errorf("failed to change back to root directory: %v", err)
		}
	}

	// Create graph
	graph := &Graph{
		Modules: modules,
	}

	// First pass: Create all nodes and build module prefix map
	moduleMap := make(map[string]bool)
	for _, mod := range modules {
		moduleMap[mod.ModulePath] = true
	}

	// Helper function to check if a package belongs to our modules
	isInternalPackage := func(pkgPath string) bool {
		for modPath := range moduleMap {
			if strings.HasPrefix(pkgPath, modPath) {
				return true
			}
		}
		return false
	}

	// Create nodes
	nodeMap := make(map[string]bool)
	for _, pkg := range allPackages {
		// Only create nodes for packages in our modules
		if isInternalPackage(pkg.ImportPath) {
			if !nodeMap[pkg.ImportPath] {
				nodeMap[pkg.ImportPath] = true

				// Find which module this package belongs to
				var moduleName string
				for _, mod := range modules {
					if strings.HasPrefix(pkg.ImportPath, mod.ModulePath) {
						moduleName = mod.ModulePath
						break
					}
				}

				graph.Nodes = append(graph.Nodes, Node{
					ID:     pkg.ImportPath,
					Module: moduleName,
				})
			}
		}
	}

	// Second pass: Create links only between internal packages
	for _, pkg := range allPackages {
		if !isInternalPackage(pkg.ImportPath) {
			continue
		}

		for _, imp := range pkg.Imports {
			// Only include dependencies between our internal packages
			if isInternalPackage(imp) {
				graph.Links = append(graph.Links, Link{
					Source: pkg.ImportPath,
					Target: imp,
				})
			}
		}
	}

	return graph, nil
}

func main() {
	var ignoredPathsFlag string
	flag.StringVar(&ignoredPathsFlag, "ignore", "", "Comma-separated list of paths to ignore (relative to root directory)")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [working_directory]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGenerates a dependency graph visualization for a Go project.\n")
		fmt.Fprintf(os.Stderr, "\nArguments:\n")
		fmt.Fprintf(os.Stderr, "  working_directory    The root directory of the Go project (default: current directory)\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Parse ignored paths
	if ignoredPathsFlag != "" {
		ignoredPaths = strings.Split(ignoredPathsFlag, ",")
		// Trim spaces from paths
		for i := range ignoredPaths {
			ignoredPaths[i] = strings.TrimSpace(ignoredPaths[i])
		}
	}

	workDir := "."
	if flag.NArg() > 0 {
		workDir = flag.Arg(0)
	}

	// Convert to absolute path
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Verify directory exists
	if stat, err := os.Stat(absWorkDir); err != nil || !stat.IsDir() {
		log.Fatalf("Invalid working directory: %s", absWorkDir)
	}

	// Change to the working directory
	if err := os.Chdir(absWorkDir); err != nil {
		log.Fatalf("Failed to change to working directory: %v", err)
	}

	data, err := extractPackageDependencies()
	if err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	// Create template
	tmpl := template.Must(template.New("graph").Parse(htmlTemplate))
	
	// Create output file in the working directory
	outputPath := filepath.Join(absWorkDir, "dependency_graph.html")
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// Execute template and write to file
	if err := tmpl.Execute(f, string(jsonData)); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dependency graph has been generated in %s\n", outputPath)
}
