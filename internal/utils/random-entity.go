package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	sq "github.com/Masterminds/squirrel"
)

type Join struct {
	Type      string
	Table     string
	Condition string
}

type Option func(*randomIDOptions) error

type randomIDOptions struct {
	joins       []Join
	whereClause sq.Sqlizer
}

func WithJoins(joins []Join) Option {
	return func(opts *randomIDOptions) error {
		opts.joins = joins
		return nil
	}
}

func WithWhereClause(where sq.Sqlizer) Option {
	return func(opts *randomIDOptions) error {
		opts.whereClause = where
		return nil
	}
}

func RandomIDWithBuilder(ctx context.Context, db *pgxpool.Pool, table, column string, opts ...Option) (int, error) {

	options := &randomIDOptions{
		joins:       nil,
		whereClause: nil,
	}

	for _, opt := range opts {
		if err := opt(options); err != nil {
			return 0, fmt.Errorf("option build error: %w", err)
		}
	}
	qualifiedColumn := fmt.Sprintf("%s.%s", table, column)
	queryBuilder := sq.Select(qualifiedColumn).From(table).PlaceholderFormat(sq.Dollar)

	for _, join := range options.joins {
		joinType := strings.ToUpper(join.Type)
		switch joinType {
		case "INNER":
			queryBuilder = queryBuilder.InnerJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case "LEFT":
			queryBuilder = queryBuilder.LeftJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		case "RIGHT":
			queryBuilder = queryBuilder.RightJoin(fmt.Sprintf("%s ON %s", join.Table, join.Condition))
		default:
			return 0, fmt.Errorf("неподдерживаемый тип JOIN: %s", join.Type)
		}
	}

	if options.whereClause != nil {
		queryBuilder = queryBuilder.Where(options.whereClause)
	}

	queryBuilder = queryBuilder.OrderBy("RANDOM()").Limit(1)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("query build error: %w", err)
	}

	var id int
	err = db.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("cant find random %s: %v", column, err)
	}

	return id, nil
}
