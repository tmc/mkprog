package generator

import (
	"fmt"
	"strings"

	"github.com/awalterschulze/gographviz"
)

// GenerateGraph generates a DOT graph of the schema
func GenerateGraph(schema *Schema) (string, error) {
	g := gographviz.NewGraph()
	if err := g.SetName("Schema"); err != nil {
		return "", fmt.Errorf("failed to set graph name: %w", err)
	}

	if err := g.SetDir(true); err != nil {
		return "", fmt.Errorf("failed to set directed graph: %w", err)
	}

	// Set graph attributes
	g.AddAttr("Schema", "rankdir", "LR")
	g.AddAttr("Schema", "splines", "ortho")
	g.AddAttr("Schema", "nodesep", "0.8")
	g.AddAttr("Schema", "ranksep", "1.0")
	g.AddAttr("Schema", "fontname", "Arial")
	g.AddAttr("Schema", "fontsize", "12")

	// Set node defaults
	g.AddAttr("Schema", "node", "shape", "plaintext")
	g.AddAttr("Schema", "node", "fontname", "Arial")
	g.AddAttr("Schema", "node", "fontsize", "10")

	// Set edge defaults
	g.AddAttr("Schema", "edge", "fontname", "Arial")
	g.AddAttr("Schema", "edge", "fontsize", "8")
	g.AddAttr("Schema", "edge", "arrowsize", "0.7")

	// Create nodes for each table
	for tableName, table := range schema.Tables {
		// Create HTML-like label for table
		label := generateTableLabel(table)
		attrs := map[string]string{
			"label":     label,
			"id":        tableName,
			"shape":     "none",
			"margin":    "0",
			"fillcolor": "#E7F2FA",
			"style":     "filled",
		}
		if err := g.AddNode("Schema", tableName, attrs); err != nil {
			return "", fmt.Errorf("failed to add node for table %s: %w", tableName, err)
		}
	}

	// Create edges for foreign key relationships
	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			if col.References != "" {
				// Parse the reference (table.column)
				refParts := strings.Split(col.References, ".")
				if len(refParts) != 2 {
					continue
				}
				refTable := refParts[0]

				// Create edge attributes
				attrs := map[string]string{
					"headlabel": " ",
					"taillabel": col.Name,
					"dir":       "both",
					"arrowtail": "crow",
					"arrowhead": "tee",
					"color":     "#2471A3",
				}

				if col.OnDelete != "" {
					attrs["label"] = fmt.Sprintf(" %s ", col.OnDelete)
				}

				if err := g.AddEdge(table.Name, refTable, true, attrs); err != nil {
					return "", fmt.Errorf("failed to add edge from %s to %s: %w", table.Name, refTable, err)
				}
			}
		}
	}

	return g.String(), nil
}

// generateTableLabel creates an HTML-like label for a table node
func generateTableLabel(table *Table) string {
	var sb strings.Builder

	// Start the HTML table
	sb.WriteString("<<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"4\" BGCOLOR=\"#E7F2FA\">\n")

	// Table header
	sb.WriteString(fmt.Sprintf("<TR><TD COLSPAN=\"3\" BGCOLOR=\"#2471A3\"><FONT COLOR=\"white\"><B>%s</B></FONT></TD></TR>\n", table.Name))

	// Column headers
	sb.WriteString("<TR><TD BGCOLOR=\"#AED6F1\"><B>Column</B></TD>")
	sb.WriteString("<TD BGCOLOR=\"#AED6F1\"><B>Type</B></TD>")
	sb.WriteString("<TD BGCOLOR=\"#AED6F1\"><B>Constraints</B></TD></TR>\n")

	// Columns
	for colName, col := range table.Columns {
		// Determine background color based on whether it's a primary key
		bgColor := "#FFFFFF"
		if col.PrimaryKey {
			bgColor = "#D5F5E3"
		} else if col.References != "" {
			bgColor = "#FCF3CF"
		}

		// Column name
		sb.WriteString(fmt.Sprintf("<TR><TD BGCOLOR=\"%s\">%s</TD>\n", bgColor, colName))

		// Column type
		sb.WriteString(fmt.Sprintf("<TD BGCOLOR=\"%s\">%s", bgColor, col.Type))
		if col.Length > 0 {
			sb.WriteString(fmt.Sprintf("(%d)", col.Length))
		}
		sb.WriteString("</TD>\n")

		// Constraints
		sb.WriteString(fmt.Sprintf("<TD BGCOLOR=\"%s\">", bgColor))
		var constraints []string
		
		if col.PrimaryKey {
			constraints = append(constraints, "PK")
		}
		if col.Nullable {
			constraints = append(constraints, "NULL")
		} else {
			constraints = append(constraints, "NOT NULL")
		}
		if col.Unique {
			constraints = append(constraints, "UNIQUE")
		}
		if col.AutoIncrement {
			constraints = append(constraints, "AUTO")
		}
		if col.Default != "" {
			constraints = append(constraints, fmt.Sprintf("DEFAULT %s", col.Default))
		}
		if col.References != "" {
			constraints = append(constraints, fmt.Sprintf("FK → %s", col.References))
		}

		sb.WriteString(strings.Join(constraints, ", "))
		sb.WriteString("</TD></TR>\n")
	}

	// End the HTML table
	sb.WriteString("</TABLE>>")

	return sb.String()
}