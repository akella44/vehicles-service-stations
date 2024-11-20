package model

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AllowedTables struct {
	tables map[string][]string
}

func NewAllowedTables() *AllowedTables {
	return &AllowedTables{
		tables: make(map[string][]string),
	}
}

func (at *AllowedTables) Initialize(ctx context.Context, db *pgxpool.Pool, schema string) error {
	tablesQuery := `
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = $1
          AND table_type = 'BASE TABLE';
    `
	rows, err := db.Query(ctx, tablesQuery, schema)
	if err != nil {
		return fmt.Errorf("failed to retrieve tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tables: %w", err)
	}

	for _, table := range tables {
		columnsQuery := `
            SELECT column_name
            FROM information_schema.columns
            WHERE table_schema = $1
              AND table_name = $2;
        `
		columnsRows, err := db.Query(ctx, columnsQuery, schema, table)
		if err != nil {
			return fmt.Errorf("failed to retrieve columns for table %s: %w", table, err)
		}

		var columns []string
		for columnsRows.Next() {
			var column string
			if err := columnsRows.Scan(&column); err != nil {
				columnsRows.Close()
				return fmt.Errorf("failed to scan column for table %s: %w", table, err)
			}
			columns = append(columns, column)
		}
		columnsRows.Close()

		if err := columnsRows.Err(); err != nil {
			return fmt.Errorf("error iterating columns for table %s: %w", table, err)
		}

		at.tables[table] = columns
	}

	return nil
}

func (at *AllowedTables) IsValid(table, column string) bool {
	columns, exists := at.tables[table]
	if !exists {
		return false
	}
	if column == "" {
		return true
	}
	for _, col := range columns {
		if col == column {
			return true
		}
	}
	return false
}
