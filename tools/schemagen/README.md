# schemagen - Database Schema Generator

`schemagen` is a tool that generates database schemas and migrations from high-level descriptions. It allows developers to define database schemas using natural language or simplified syntax and generates SQL, Go structs, and migrations.

## Features

- Generate SQL schema definitions from compact descriptions
- Support for multiple database backends (PostgreSQL, MySQL, SQLite)
- Generate Go struct models with tags
- Create migration files with up/down operations
- Visualize schema relationships
- Smart recommendations for indexes and constraints

## Installation

```bash
go install github.com/tmc/mkprog/tools/schemagen@latest
```

## Usage

```bash
# Generate schema from description
schemagen --db postgres "users(id:uuid,name:text,email:text:unique,created_at:timestamp)" > schema.sql

# Generate Go models
schemagen --lang go --db postgres "users(id:uuid,name:text,email:text:unique,created_at:timestamp)" > models.go

# Generate from a schema file
schemagen --input schema.yaml --output-dir ./db

# Visualize schema
schemagen --visualize --input schema.yaml > schema.dot
dot -Tpng schema.dot -o schema.png

# Generate migrations
schemagen --migrations --from schema_v1.yaml --to schema_v2.yaml
```

## Schema Format

You can use a simplified syntax for quick schema definitions:

```
table_name(column:type[:constraint], ...)
```

Or use YAML for more complex schemas:

```yaml
tables:
  users:
    columns:
      id:
        type: uuid
        primary_key: true
      name:
        type: text
        nullable: false
      email:
        type: text
        unique: true
      created_at:
        type: timestamp
        default: "NOW()"
    indexes:
      - name: users_email_idx
        columns: [email]
    
  posts:
    columns:
      id:
        type: uuid
        primary_key: true
      user_id:
        type: uuid
        references: users.id
      title:
        type: text
      body:
        type: text
      published:
        type: boolean
        default: false
```

## License

MIT