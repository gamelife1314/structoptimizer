package reporter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gamelife1314/structoptimizer/optimizer"
)

// hierarchyNode represents a node in the struct hierarchy tree
type hierarchyNode struct {
	Key      string
	Name     string
	Depth    int
	OrigSize int64
	OptSize  int64
	Saved    int64
	Skipped  bool
	Children []*hierarchyNode
}

// buildHierarchyTree builds a tree from struct reports
func buildHierarchyTree(reports []*optimizer.StructReport) []*hierarchyNode {
	nodeMap := make(map[string]*hierarchyNode)
	var roots []*hierarchyNode

	for _, sr := range reports {
		key := sr.PkgPath + "." + sr.Name
		node := &hierarchyNode{
			Key:      key,
			Name:     sr.Name,
			Depth:    sr.Depth,
			OrigSize: sr.OrigSize,
			OptSize:  sr.OptSize,
			Saved:    sr.Saved,
			Skipped:  sr.Skipped,
		}
		nodeMap[key] = node

		if sr.ParentKey == "" {
			roots = append(roots, node)
		}
	}

	for _, sr := range reports {
		if sr.ParentKey == "" {
			continue
		}
		key := sr.PkgPath + "." + sr.Name
		if parent, ok := nodeMap[sr.ParentKey]; ok {
			if child, ok := nodeMap[key]; ok {
				parent.Children = append(parent.Children, child)
			}
		}
	}

	// Sort children by name for deterministic output
	var sortChildren func(nodes []*hierarchyNode)
	sortChildren = func(nodes []*hierarchyNode) {
		sort.SliceStable(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})
		for _, n := range nodes {
			sortChildren(n.Children)
		}
	}
	sortChildren(roots)

	return roots
}

// renderHierarchyTXT renders the hierarchy as TXT
func renderHierarchyTXT(sb *strings.Builder, roots []*hierarchyNode, prefix string, isLast bool, label string) {
	for i, node := range roots {
		isNodeLast := i == len(roots)-1
		connector := "├── "
		childPrefix := "│   "
		if isNodeLast {
			connector = "└── "
			childPrefix = "    "
		}

		info := ""
		if node.Skipped {
			info = " [skipped]"
		} else if node.Saved > 0 {
			info = fmt.Sprintf(" (%d→%d, saved:%d)", node.OrigSize, node.OptSize, node.Saved)
		} else {
			info = fmt.Sprintf(" (%d bytes)", node.OrigSize)
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, connector, node.Name, info))

		if len(node.Children) > 0 {
			renderHierarchyTXT(sb, node.Children, prefix+childPrefix, isNodeLast, label)
		}
	}
}

// renderHierarchyMD renders the hierarchy as Markdown
func renderHierarchyMD(sb *strings.Builder, roots []*hierarchyNode, prefix string, isLast bool) {
	for i, node := range roots {
		isNodeLast := i == len(roots)-1
		connector := "├── "
		childPrefix := "│   "
		if isNodeLast {
			connector = "└── "
			childPrefix = "    "
		}

		info := ""
		if node.Skipped {
			info = " *(skipped)*"
		} else if node.Saved > 0 {
			info = fmt.Sprintf(" *(optimized: %d→%d, saved: %d bytes)*", node.OrigSize, node.OptSize, node.Saved)
		} else {
			info = fmt.Sprintf(" *(%d bytes)*", node.OrigSize)
		}

		sb.WriteString(fmt.Sprintf("`%s`%s**%s**%s\n", prefix+connector, "", node.Name, info))

		if len(node.Children) > 0 {
			renderHierarchyMD(sb, node.Children, prefix+childPrefix, isNodeLast)
		}
	}
}

// renderHierarchyHTML renders the hierarchy as HTML
func renderHierarchyHTML(sb *strings.Builder, roots []*hierarchyNode, prefix string, isLast bool) {
	for i, node := range roots {
		isNodeLast := i == len(roots)-1
		connector := "├── "
		childPrefix := "│   "
		if isNodeLast {
			connector = "└── "
			childPrefix = "    "
		}

		info := ""
		if node.Skipped {
			info = " <em>(skipped)</em>"
		} else if node.Saved > 0 {
			info = fmt.Sprintf(" <em>(%d→%d, saved: %d bytes)</em>", node.OrigSize, node.OptSize, node.Saved)
		} else {
			info = fmt.Sprintf(" <em>(%d bytes)</em>", node.OrigSize)
		}

		sb.WriteString(fmt.Sprintf("%s<strong>%s</strong>%s<br>\n", prefix+connector, node.Name, info))

		if len(node.Children) > 0 {
			renderHierarchyHTML(sb, node.Children, prefix+childPrefix, isNodeLast)
		}
	}
}

// addHierarchySection adds the hierarchy tree to the report if in -struct mode
func addHierarchySectionTXT(sb *strings.Builder, report *optimizer.Report) {
	if report.RootStruct == "" || len(report.StructReports) == 0 {
		return
	}
	roots := buildHierarchyTree(report.StructReports)
	if len(roots) == 0 {
		return
	}

	sb.WriteString("┌────────────────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│  🌳 Struct Hierarchy%58s│\n", ""))
	sb.WriteString("└────────────────────────────────────────────────────────────────────────────────┘\n")
	renderHierarchyTXT(sb, roots, "", true, "")
	sb.WriteString("\n")
}

func addHierarchySectionMD(sb *strings.Builder, report *optimizer.Report) {
	if report.RootStruct == "" || len(report.StructReports) == 0 {
		return
	}
	roots := buildHierarchyTree(report.StructReports)
	if len(roots) == 0 {
		return
	}

	sb.WriteString("## 🌳 Struct Hierarchy\n\n")
	sb.WriteString("<pre>\n")
	renderHierarchyMD(sb, roots, "", true)
	sb.WriteString("</pre>\n\n")
}

func addHierarchySectionHTML(sb *strings.Builder, report *optimizer.Report) {
	if report.RootStruct == "" || len(report.StructReports) == 0 {
		return
	}
	roots := buildHierarchyTree(report.StructReports)
	if len(roots) == 0 {
		return
	}

	sb.WriteString("<h2>🌳 Struct Hierarchy</h2>\n")
	sb.WriteString("<pre class=\"hierarchy\">\n")
	renderHierarchyHTML(sb, roots, "", true)
	sb.WriteString("</pre>\n")
}
