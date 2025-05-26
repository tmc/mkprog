package generator

import (
	"fmt"
	"strings"
)

// GenerateMigration generates SQL migration between two schemas
func GenerateMigration(fromSchema, toSchema *Schema, dbType string) (*Migration, error) {
	switch dbType {
	case "postgres", "postgresql":
		return generatePostgresMigration(fromSchema, toSchema)
	case "mysql":
		return generateMySQLMigration(fromSchema, toSchema)
	case "sqlite":
		return generateSQLiteMigration(fromSchema, toSchema)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// generatePostgresMigration generates PostgreSQL migrations
func generatePostgresMigration(fromSchema, toSchema *Schema) (*Migration, error) {
	var upBuilder, downBuilder strings.Builder

	upBuilder.WriteString("-- Migration Up\n\n")
	downBuilder.WriteString("-- Migration Down\n\n")

	// Tables to add (exist in toSchema but not in fromSchema)
	// Tables to remove (exist in fromSchema but not in toSchema)
	for tableName, toTable := range toSchema.Tables {
		if _, exists := fromSchema.Tables[tableName]; !exists {
			// New table - add it in the up migration
			generateCreateTableSQL(&upBuilder, toTable, "postgres")
			
			// Add drop table to down migration
			downBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;\n\n", tableName))
		}
	}

	// Tables to remove
	for tableName := range fromSchema.Tables {
		if _, exists := toSchema.Tables[tableName]; !exists {
			// Table removed - drop it in the up migration
			upBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;\n\n", tableName))
			
			// Add create table to down migration
			generateCreateTableSQL(&downBuilder, fromSchema.Tables[tableName], "postgres")
		}
	}

	// Tables to alter (exist in both schemas but have differences)
	for tableName, fromTable := range fromSchema.Tables {
		toTable, exists := toSchema.Tables[tableName]
		if !exists {
			continue // Table was removed, already handled
		}

		// Check for column changes
		alterTableUp := fmt.Sprintf("ALTER TABLE %s\n", tableName)
		alterTableDown := fmt.Sprintf("ALTER TABLE %s\n", tableName)
		
		hasChanges := false
		
		// Columns to add
		for colName, toCol := range toTable.Columns {
			if _, exists := fromTable.Columns[colName]; !exists {
				// New column - add it
				hasChanges = true
				
				// Up migration - add column
				upBuilder.WriteString(alterTableUp)
				upBuilder.WriteString(fmt.Sprintf("  ADD COLUMN %s %s", colName, postgresDataType(toCol)))
				if !toCol.Nullable {
					upBuilder.WriteString(" NOT NULL")
				}
				if toCol.Default != "" {
					upBuilder.WriteString(fmt.Sprintf(" DEFAULT %s", toCol.Default))
				}
				upBuilder.WriteString(";\n\n")
				
				// Down migration - drop column
				downBuilder.WriteString(alterTableDown)
				downBuilder.WriteString(fmt.Sprintf("  DROP COLUMN %s;\n\n", colName))
			}
		}
		
		// Columns to remove
		for colName, fromCol := range fromTable.Columns {
			if _, exists := toTable.Columns[colName]; !exists {
				// Column removed - drop it
				hasChanges = true
				
				// Up migration - drop column
				upBuilder.WriteString(alterTableUp)
				upBuilder.WriteString(fmt.Sprintf("  DROP COLUMN %s;\n\n", colName))
				
				// Down migration - add column back
				downBuilder.WriteString(alterTableDown)
				downBuilder.WriteString(fmt.Sprintf("  ADD COLUMN %s %s", colName, postgresDataType(fromCol)))
				if !fromCol.Nullable {
					downBuilder.WriteString(" NOT NULL")
				}
				if fromCol.Default != "" {
					downBuilder.WriteString(fmt.Sprintf(" DEFAULT %s", fromCol.Default))
				}
				downBuilder.WriteString(";\n\n")
			}
		}
		
		// Columns to alter (type changes, constraints, etc.)
		for colName, fromCol := range fromTable.Columns {
			toCol, exists := toTable.Columns[colName]
			if !exists {
				continue // Column was removed, already handled
			}
			
			// Check for type changes
			if fromCol.Type != toCol.Type {
				hasChanges = true
				
				// Up migration - alter column type
				upBuilder.WriteString(alterTableUp)
				upBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s TYPE %s", colName, postgresDataType(toCol)))
				if toCol.Length > 0 {
					upBuilder.WriteString(fmt.Sprintf("(%d)", toCol.Length))
				}
				upBuilder.WriteString(";\n\n")
				
				// Down migration - restore column type
				downBuilder.WriteString(alterTableDown)
				downBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s TYPE %s", colName, postgresDataType(fromCol)))
				if fromCol.Length > 0 {
					downBuilder.WriteString(fmt.Sprintf("(%d)", fromCol.Length))
				}
				downBuilder.WriteString(";\n\n")
			}
			
			// Check for nullable changes
			if fromCol.Nullable != toCol.Nullable {
				hasChanges = true
				
				if toCol.Nullable {
					// Up migration - drop not null
					upBuilder.WriteString(alterTableUp)
					upBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s DROP NOT NULL;\n\n", colName))
					
					// Down migration - add not null
					downBuilder.WriteString(alterTableDown)
					downBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s SET NOT NULL;\n\n", colName))
				} else {
					// Up migration - add not null
					upBuilder.WriteString(alterTableUp)
					upBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s SET NOT NULL;\n\n", colName))
					
					// Down migration - drop not null
					downBuilder.WriteString(alterTableDown)
					downBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s DROP NOT NULL;\n\n", colName))
				}
			}
			
			// Check for default changes
			if fromCol.Default != toCol.Default {
				hasChanges = true
				
				if toCol.Default == "" {
					// Up migration - drop default
					upBuilder.WriteString(alterTableUp)
					upBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s DROP DEFAULT;\n\n", colName))
				} else {
					// Up migration - set default
					upBuilder.WriteString(alterTableUp)
					upBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s SET DEFAULT %s;\n\n", colName, toCol.Default))
				}
				
				if fromCol.Default == "" {
					// Down migration - drop default
					downBuilder.WriteString(alterTableDown)
					downBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s DROP DEFAULT;\n\n", colName))
				} else {
					// Down migration - set default
					downBuilder.WriteString(alterTableDown)
					downBuilder.WriteString(fmt.Sprintf("  ALTER COLUMN %s SET DEFAULT %s;\n\n", colName, fromCol.Default))
				}
			}
		}
		
		// Handle index changes
		// New indexes
		for _, toIdx := range toTable.Indexes {
			found := false
			for _, fromIdx := range fromTable.Indexes {
				if fromIdx.Name == toIdx.Name {
					found = true
					break
				}
			}
			
			if !found {
				// New index - create it in up migration
				idxType := "INDEX"
				if toIdx.Unique {
					idxType = "UNIQUE INDEX"
				}
				
				upBuilder.WriteString(fmt.Sprintf("CREATE %s %s ON %s (%s);\n\n",
					idxType, toIdx.Name, tableName, strings.Join(toIdx.Columns, ", ")))
				
				// Drop in down migration
				downBuilder.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS %s;\n\n", toIdx.Name))
			}
		}
		
		// Removed indexes
		for _, fromIdx := range fromTable.Indexes {
			found := false
			for _, toIdx := range toTable.Indexes {
				if toIdx.Name == fromIdx.Name {
					found = true
					break
				}
			}
			
			if !found {
				// Index removed - drop it in up migration
				upBuilder.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS %s;\n\n", fromIdx.Name))
				
				// Recreate in down migration
				idxType := "INDEX"
				if fromIdx.Unique {
					idxType = "UNIQUE INDEX"
				}
				
				downBuilder.WriteString(fmt.Sprintf("CREATE %s %s ON %s (%s);\n\n",
					idxType, fromIdx.Name, tableName, strings.Join(fromIdx.Columns, ", ")))
			}
		}
	}

	return &Migration{
		Up:   upBuilder.String(),
		Down: downBuilder.String(),
	}, nil
}

// generateMySQLMigration generates MySQL migrations
func generateMySQLMigration(fromSchema, toSchema *Schema) (*Migration, error) {
	var upBuilder, downBuilder strings.Builder

	upBuilder.WriteString("-- Migration Up\n\n")
	downBuilder.WriteString("-- Migration Down\n\n")

	// Handle table additions
	for tableName, toTable := range toSchema.Tables {
		if _, exists := fromSchema.Tables[tableName]; !exists {
			// New table - add it in the up migration
			generateCreateTableSQL(&upBuilder, toTable, "mysql")
			
			// Add drop table to down migration
			downBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n\n", tableName))
		}
	}

	// Handle table removals
	for tableName := range fromSchema.Tables {
		if _, exists := toSchema.Tables[tableName]; !exists {
			// Table removed - drop it in the up migration
			upBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n\n", tableName))
			
			// Add create table to down migration
			generateCreateTableSQL(&downBuilder, fromSchema.Tables[tableName], "mysql")
		}
	}

	// Handle table alterations
	for tableName, fromTable := range fromSchema.Tables {
		toTable, exists := toSchema.Tables[tableName]
		if !exists {
			continue // Table was removed, already handled
		}

		// Check for column changes
		alterTableUp := fmt.Sprintf("ALTER TABLE `%s`\n", tableName)
		alterTableDown := fmt.Sprintf("ALTER TABLE `%s`\n", tableName)
		
		hasChangesUp := false
		hasChangesDown := false
		
		// Columns to add
		for colName, toCol := range toTable.Columns {
			if _, exists := fromTable.Columns[colName]; !exists {
				// New column - add it
				if hasChangesUp {
					alterTableUp += ",\n"
				}
				hasChangesUp = true
				
				// Up migration - add column
				alterTableUp += fmt.Sprintf("  ADD COLUMN `%s` %s", colName, mysqlDataType(toCol))
				if !toCol.Nullable {
					alterTableUp += " NOT NULL"
				}
				if toCol.Default != "" {
					alterTableUp += fmt.Sprintf(" DEFAULT %s", toCol.Default)
				}
				if toCol.AutoIncrement {
					alterTableUp += " AUTO_INCREMENT"
				}
				
				// Down migration - drop column
				if hasChangesDown {
					alterTableDown += ",\n"
				}
				hasChangesDown = true
				alterTableDown += fmt.Sprintf("  DROP COLUMN `%s`", colName)
			}
		}
		
		// Columns to remove
		for colName, fromCol := range fromTable.Columns {
			if _, exists := toTable.Columns[colName]; !exists {
				// Column removed - drop it
				if hasChangesUp {
					alterTableUp += ",\n"
				}
				hasChangesUp = true
				
				// Up migration - drop column
				alterTableUp += fmt.Sprintf("  DROP COLUMN `%s`", colName)
				
				// Down migration - add column back
				if hasChangesDown {
					alterTableDown += ",\n"
				}
				hasChangesDown = true
				alterTableDown += fmt.Sprintf("  ADD COLUMN `%s` %s", colName, mysqlDataType(fromCol))
				if !fromCol.Nullable {
					alterTableDown += " NOT NULL"
				}
				if fromCol.Default != "" {
					alterTableDown += fmt.Sprintf(" DEFAULT %s", fromCol.Default)
				}
				if fromCol.AutoIncrement {
					alterTableDown += " AUTO_INCREMENT"
				}
			}
		}
		
		// Columns to modify
		for colName, fromCol := range fromTable.Columns {
			toCol, exists := toTable.Columns[colName]
			if !exists {
				continue // Column was removed, already handled
			}
			
			// Check if there are any differences
			if fromCol.Type != toCol.Type || 
			   fromCol.Nullable != toCol.Nullable || 
			   fromCol.Default != toCol.Default ||
			   fromCol.AutoIncrement != toCol.AutoIncrement {
				
				if hasChangesUp {
					alterTableUp += ",\n"
				}
				hasChangesUp = true
				
				// Up migration - modify column
				alterTableUp += fmt.Sprintf("  MODIFY COLUMN `%s` %s", colName, mysqlDataType(toCol))
				if !toCol.Nullable {
					alterTableUp += " NOT NULL"
				}
				if toCol.Default != "" {
					alterTableUp += fmt.Sprintf(" DEFAULT %s", toCol.Default)
				}
				if toCol.AutoIncrement {
					alterTableUp += " AUTO_INCREMENT"
				}
				
				// Down migration - restore column
				if hasChangesDown {
					alterTableDown += ",\n"
				}
				hasChangesDown = true
				alterTableDown += fmt.Sprintf("  MODIFY COLUMN `%s` %s", colName, mysqlDataType(fromCol))
				if !fromCol.Nullable {
					alterTableDown += " NOT NULL"
				}
				if fromCol.Default != "" {
					alterTableDown += fmt.Sprintf(" DEFAULT %s", fromCol.Default)
				}
				if fromCol.AutoIncrement {
					alterTableDown += " AUTO_INCREMENT"
				}
			}
		}
		
		// Add ALTER TABLE statements if there were changes
		if hasChangesUp {
			upBuilder.WriteString(alterTableUp + ";\n\n")
		}
		
		if hasChangesDown {
			downBuilder.WriteString(alterTableDown + ";\n\n")
		}
		
		// Handle index changes
		// New indexes
		for _, toIdx := range toTable.Indexes {
			found := false
			for _, fromIdx := range fromTable.Indexes {
				if fromIdx.Name == toIdx.Name {
					found = true
					break
				}
			}
			
			if !found {
				// New index - create it in up migration
				idxType := "INDEX"
				if toIdx.Unique {
					idxType = "UNIQUE INDEX"
				}
				
				upBuilder.WriteString(fmt.Sprintf("CREATE %s `%s` ON `%s` (`%s`);\n\n",
					idxType, toIdx.Name, tableName, strings.Join(toIdx.Columns, "`, `")))
				
				// Drop in down migration
				downBuilder.WriteString(fmt.Sprintf("DROP INDEX `%s` ON `%s`;\n\n", 
					toIdx.Name, tableName))
			}
		}
		
		// Removed indexes
		for _, fromIdx := range fromTable.Indexes {
			found := false
			for _, toIdx := range toTable.Indexes {
				if toIdx.Name == fromIdx.Name {
					found = true
					break
				}
			}
			
			if !found {
				// Index removed - drop it in up migration
				upBuilder.WriteString(fmt.Sprintf("DROP INDEX `%s` ON `%s`;\n\n", 
					fromIdx.Name, tableName))
				
				// Recreate in down migration
				idxType := "INDEX"
				if fromIdx.Unique {
					idxType = "UNIQUE INDEX"
				}
				
				downBuilder.WriteString(fmt.Sprintf("CREATE %s `%s` ON `%s` (`%s`);\n\n",
					idxType, fromIdx.Name, tableName, strings.Join(fromIdx.Columns, "`, `")))
			}
		}
	}

	return &Migration{
		Up:   upBuilder.String(),
		Down: downBuilder.String(),
	}, nil
}

// generateSQLiteMigration generates SQLite migrations
// Note: SQLite has limited ALTER TABLE support, so we use a different approach
func generateSQLiteMigration(fromSchema, toSchema *Schema) (*Migration, error) {
	var upBuilder, downBuilder strings.Builder

	upBuilder.WriteString("-- Migration Up\n\n")
	downBuilder.WriteString("-- Migration Down\n\n")

	// For SQLite, we'll use a simple approach of just dropping and recreating tables
	// that have changes, since SQLite's ALTER TABLE is limited
	
	// Handle table additions
	for tableName, toTable := range toSchema.Tables {
		if _, exists := fromSchema.Tables[tableName]; !exists {
			// New table - add it in the up migration
			generateCreateTableSQL(&upBuilder, toTable, "sqlite")
			
			// Add drop table to down migration
			downBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n\n", tableName))
		}
	}

	// Handle table removals
	for tableName := range fromSchema.Tables {
		if _, exists := toSchema.Tables[tableName]; !exists {
			// Table removed - drop it in the up migration
			upBuilder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n\n", tableName))
			
			// Add create table to down migration
			generateCreateTableSQL(&downBuilder, fromSchema.Tables[tableName], "sqlite")
		}
	}

	// For tables that exist in both schemas but have changes, use the recreate table approach
	for tableName, fromTable := range fromSchema.Tables {
		toTable, exists := toSchema.Tables[tableName]
		if !exists {
			continue // Table was removed, already handled
		}

		// Check if there are any differences
		if !tablesEqual(fromTable, toTable) {
			// Tables are different, use the recreate approach
			upBuilder.WriteString(fmt.Sprintf("-- Recreating table %s\n", tableName))
			upBuilder.WriteString(fmt.Sprintf("PRAGMA foreign_keys=off;\n"))
			upBuilder.WriteString(fmt.Sprintf("BEGIN TRANSACTION;\n\n"))
			
			// Create new table with temporary name
			tempTableName := tableName + "_new"
			toTable.Name = tempTableName
			generateCreateTableSQL(&upBuilder, toTable, "sqlite")
			
			// Copy data from old table to new table
			// We need to match columns that exist in both tables
			var commonCols []string
			for colName := range toTable.Columns {
				if _, exists := fromTable.Columns[colName]; exists {
					commonCols = append(commonCols, colName)
				}
			}
			
			upBuilder.WriteString(fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s;\n\n",
				tempTableName, strings.Join(commonCols, ", "), strings.Join(commonCols, ", "), tableName))
			
			// Drop old table and rename new table
			upBuilder.WriteString(fmt.Sprintf("DROP TABLE %s;\n", tableName))
			upBuilder.WriteString(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;\n\n", tempTableName, tableName))
			
			// Recreate indexes
			for _, idx := range toTable.Indexes {
				idxType := ""
				if idx.Unique {
					idxType = "UNIQUE "
				}
				
				upBuilder.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);\n",
					idxType, idx.Name, tableName, strings.Join(idx.Columns, ", ")))
			}
			
			upBuilder.WriteString(fmt.Sprintf("COMMIT;\n"))
			upBuilder.WriteString(fmt.Sprintf("PRAGMA foreign_keys=on;\n\n"))
			
			// For down migration, do the same but in reverse
			downBuilder.WriteString(fmt.Sprintf("-- Reverting table %s\n", tableName))
			downBuilder.WriteString(fmt.Sprintf("PRAGMA foreign_keys=off;\n"))
			downBuilder.WriteString(fmt.Sprintf("BEGIN TRANSACTION;\n\n"))
			
			// Create new table with temporary name
			tempTableName = tableName + "_old"
			fromTable.Name = tempTableName
			generateCreateTableSQL(&downBuilder, fromTable, "sqlite")
			
			// Copy data from current table to old table
			// We need to match columns that exist in both tables
			commonCols = make([]string, 0)
			for colName := range fromTable.Columns {
				if _, exists := toTable.Columns[colName]; exists {
					commonCols = append(commonCols, colName)
				}
			}
			
			downBuilder.WriteString(fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s;\n\n",
				tempTableName, strings.Join(commonCols, ", "), strings.Join(commonCols, ", "), tableName))
			
			// Drop current table and rename old table
			downBuilder.WriteString(fmt.Sprintf("DROP TABLE %s;\n", tableName))
			downBuilder.WriteString(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;\n\n", tempTableName, tableName))
			
			// Recreate original indexes
			for _, idx := range fromTable.Indexes {
				idxType := ""
				if idx.Unique {
					idxType = "UNIQUE "
				}
				
				downBuilder.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);\n",
					idxType, idx.Name, tableName, strings.Join(idx.Columns, ", ")))
			}
			
			downBuilder.WriteString(fmt.Sprintf("COMMIT;\n"))
			downBuilder.WriteString(fmt.Sprintf("PRAGMA foreign_keys=on;\n\n"))
		}
	}

	return &Migration{
		Up:   upBuilder.String(),
		Down: downBuilder.String(),
	}, nil
}

// generateCreateTableSQL writes SQL to create a table to the given builder
func generateCreateTableSQL(b *strings.Builder, table *Table, dbType string) {
	switch dbType {
	case "postgres", "postgresql":
		b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

		// Columns
		columns := make([]string, 0, len(table.Columns))
		for _, col := range table.Columns {
			colDef := fmt.Sprintf("  %s %s", col.Name, postgresDataType(col))

			if col.PrimaryKey {
				colDef += " PRIMARY KEY"
			}
			if col.Unique {
				colDef += " UNIQUE"
			}
			if !col.Nullable {
				colDef += " NOT NULL"
			}
			if col.Default != "" {
				colDef += fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			if col.AutoIncrement {
				// In PostgreSQL, we typically use SERIAL types for auto-increment
				colDef = fmt.Sprintf("  %s SERIAL", col.Name)
				if col.PrimaryKey {
					colDef += " PRIMARY KEY"
				}
			}

			columns = append(columns, colDef)
		}

		// Foreign keys
		for _, col := range table.Columns {
			if col.References != "" {
				fkDef := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s", col.Name, col.References)
				if col.OnDelete != "" {
					fkDef += fmt.Sprintf(" ON DELETE %s", col.OnDelete)
				}
				if col.OnUpdate != "" {
					fkDef += fmt.Sprintf(" ON UPDATE %s", col.OnUpdate)
				}
				columns = append(columns, fkDef)
			}
		}

		b.WriteString(strings.Join(columns, ",\n"))
		b.WriteString("\n);\n\n")

		// Create indexes
		for _, idx := range table.Indexes {
			idxType := "INDEX"
			if idx.Unique {
				idxType = "UNIQUE INDEX"
			}

			method := ""
			if idx.Method != "" {
				method = fmt.Sprintf(" USING %s", idx.Method)
			}

			includes := ""
			if len(idx.Includes) > 0 {
				includes = fmt.Sprintf(" INCLUDE (%s)", strings.Join(idx.Includes, ", "))
			}

			b.WriteString(fmt.Sprintf("CREATE %s %s ON %s%s (%s)%s;\n",
				idxType, idx.Name, table.Name, method, strings.Join(idx.Columns, ", "), includes))
		}

		b.WriteString("\n")
		
	case "mysql":
		b.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", table.Name))

		// Columns
		columns := make([]string, 0, len(table.Columns))
		primaryKeys := make([]string, 0)

		for _, col := range table.Columns {
			colDef := fmt.Sprintf("  `%s` %s", col.Name, mysqlDataType(col))

			if !col.Nullable {
				colDef += " NOT NULL"
			}
			if col.Default != "" {
				colDef += fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			if col.AutoIncrement {
				colDef += " AUTO_INCREMENT"
			}
			if col.Comment != "" {
				colDef += fmt.Sprintf(" COMMENT '%s'", escapeSQL(col.Comment))
			}

			columns = append(columns, colDef)

			// Collect primary keys
			if col.PrimaryKey {
				primaryKeys = append(primaryKeys, col.Name)
			}
		}

		// Add primary key constraint if needed
		if len(primaryKeys) > 0 {
			pkDef := fmt.Sprintf("  PRIMARY KEY (`%s`)", strings.Join(primaryKeys, "`, `"))
			columns = append(columns, pkDef)
		}

		// Add unique constraints
		for _, col := range table.Columns {
			if col.Unique && !col.PrimaryKey {
				columns = append(columns, fmt.Sprintf("  UNIQUE KEY `%s_unique` (`%s`)", col.Name, col.Name))
			}
		}

		// Foreign keys
		for _, col := range table.Columns {
			if col.References != "" {
				// Parse the reference (table.column)
				refParts := strings.Split(col.References, ".")
				if len(refParts) != 2 {
					continue
				}
				refTable := refParts[0]
				refCol := refParts[1]

				fkName := fmt.Sprintf("fk_%s_%s", table.Name, col.Name)
				fkDef := fmt.Sprintf("  CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
					fkName, col.Name, refTable, refCol)

				if col.OnDelete != "" {
					fkDef += fmt.Sprintf(" ON DELETE %s", col.OnDelete)
				}
				if col.OnUpdate != "" {
					fkDef += fmt.Sprintf(" ON UPDATE %s", col.OnUpdate)
				}
				columns = append(columns, fkDef)
			}
		}

		b.WriteString(strings.Join(columns, ",\n"))

		// Table options
		engine := "InnoDB"
		charset := "utf8mb4"
		collate := "utf8mb4_unicode_ci"
		
		if props := table.Properties; props != nil {
			if e, ok := props["engine"]; ok {
				engine = e
			}
			if c, ok := props["charset"]; ok {
				charset = c
			}
			if coll, ok := props["collate"]; ok {
				collate = coll
			}
		}

		comment := ""
		if table.Comment != "" {
			comment = fmt.Sprintf(" COMMENT='%s'", escapeSQL(table.Comment))
		}

		b.WriteString(fmt.Sprintf("\n) ENGINE=%s DEFAULT CHARSET=%s COLLATE=%s%s;\n\n",
			engine, charset, collate, comment))

		// Create indexes
		for _, idx := range table.Indexes {
			idxType := "INDEX"
			if idx.Unique {
				idxType = "UNIQUE INDEX"
			}

			b.WriteString(fmt.Sprintf("CREATE %s `%s` ON `%s` (`%s`);\n",
				idxType, idx.Name, table.Name, strings.Join(idx.Columns, "`, `")))
		}

		b.WriteString("\n")
		
	case "sqlite":
		b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", table.Name))

		// Columns
		columns := make([]string, 0, len(table.Columns))
		for _, col := range table.Columns {
			colDef := fmt.Sprintf("  %s %s", col.Name, sqliteDataType(col))

			if col.PrimaryKey {
				colDef += " PRIMARY KEY"
				if col.AutoIncrement {
					colDef += " AUTOINCREMENT"
				}
			}
			if !col.Nullable {
				colDef += " NOT NULL"
			}
			if col.Unique && !col.PrimaryKey {
				colDef += " UNIQUE"
			}
			if col.Default != "" {
				colDef += fmt.Sprintf(" DEFAULT %s", col.Default)
			}

			columns = append(columns, colDef)
		}

		// Foreign keys
		for _, col := range table.Columns {
			if col.References != "" {
				fkDef := fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s", col.Name, col.References)
				if col.OnDelete != "" {
					fkDef += fmt.Sprintf(" ON DELETE %s", col.OnDelete)
				}
				if col.OnUpdate != "" {
					fkDef += fmt.Sprintf(" ON UPDATE %s", col.OnUpdate)
				}
				columns = append(columns, fkDef)
			}
		}

		b.WriteString(strings.Join(columns, ",\n"))
		b.WriteString("\n);\n\n")

		// Create indexes
		for _, idx := range table.Indexes {
			idxType := ""
			if idx.Unique {
				idxType = "UNIQUE "
			}

			b.WriteString(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);\n",
				idxType, idx.Name, table.Name, strings.Join(idx.Columns, ", ")))
		}

		b.WriteString("\n")
	}
}

// tablesEqual checks if two tables are equal
func tablesEqual(t1, t2 *Table) bool {
	// Check if the number of columns is the same
	if len(t1.Columns) != len(t2.Columns) {
		return false
	}

	// Check if all columns in t1 exist in t2 and are the same
	for colName, col1 := range t1.Columns {
		col2, exists := t2.Columns[colName]
		if !exists {
			return false
		}

		if col1.Type != col2.Type ||
			col1.Length != col2.Length ||
			col1.Precision != col2.Precision ||
			col1.Scale != col2.Scale ||
			col1.Nullable != col2.Nullable ||
			col1.Default != col2.Default ||
			col1.PrimaryKey != col2.PrimaryKey ||
			col1.Unique != col2.Unique ||
			col1.AutoIncrement != col2.AutoIncrement ||
			col1.References != col2.References {
			return false
		}
	}

	// Check if the number of indexes is the same
	if len(t1.Indexes) != len(t2.Indexes) {
		return false
	}

	// Check if all indexes in t1 exist in t2 and are the same
	for _, idx1 := range t1.Indexes {
		found := false
		for _, idx2 := range t2.Indexes {
			if idx1.Name == idx2.Name {
				found = true
				if idx1.Unique != idx2.Unique {
					return false
				}
				if len(idx1.Columns) != len(idx2.Columns) {
					return false
				}
				for i, col := range idx1.Columns {
					if col != idx2.Columns[i] {
						return false
					}
				}
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}