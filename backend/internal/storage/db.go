// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.articlesStmt, err = db.PrepareContext(ctx, articles); err != nil {
		return nil, fmt.Errorf("error preparing query Articles: %w", err)
	}
	if q.attachArticleImageStmt, err = db.PrepareContext(ctx, attachArticleImage); err != nil {
		return nil, fmt.Errorf("error preparing query AttachArticleImage: %w", err)
	}
	if q.getArticleByIDStmt, err = db.PrepareContext(ctx, getArticleByID); err != nil {
		return nil, fmt.Errorf("error preparing query GetArticleByID: %w", err)
	}
	if q.getArticleIDByTitleAndOriginStmt, err = db.PrepareContext(ctx, getArticleIDByTitleAndOrigin); err != nil {
		return nil, fmt.Errorf("error preparing query GetArticleIDByTitleAndOrigin: %w", err)
	}
	if q.newArticleStmt, err = db.PrepareContext(ctx, newArticle); err != nil {
		return nil, fmt.Errorf("error preparing query NewArticle: %w", err)
	}
	if q.newImageStmt, err = db.PrepareContext(ctx, newImage); err != nil {
		return nil, fmt.Errorf("error preparing query NewImage: %w", err)
	}
	if q.updateArticleStatsStmt, err = db.PrepareContext(ctx, updateArticleStats); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateArticleStats: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.articlesStmt != nil {
		if cerr := q.articlesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing articlesStmt: %w", cerr)
		}
	}
	if q.attachArticleImageStmt != nil {
		if cerr := q.attachArticleImageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing attachArticleImageStmt: %w", cerr)
		}
	}
	if q.getArticleByIDStmt != nil {
		if cerr := q.getArticleByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getArticleByIDStmt: %w", cerr)
		}
	}
	if q.getArticleIDByTitleAndOriginStmt != nil {
		if cerr := q.getArticleIDByTitleAndOriginStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getArticleIDByTitleAndOriginStmt: %w", cerr)
		}
	}
	if q.newArticleStmt != nil {
		if cerr := q.newArticleStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing newArticleStmt: %w", cerr)
		}
	}
	if q.newImageStmt != nil {
		if cerr := q.newImageStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing newImageStmt: %w", cerr)
		}
	}
	if q.updateArticleStatsStmt != nil {
		if cerr := q.updateArticleStatsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateArticleStatsStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                               DBTX
	tx                               *sql.Tx
	articlesStmt                     *sql.Stmt
	attachArticleImageStmt           *sql.Stmt
	getArticleByIDStmt               *sql.Stmt
	getArticleIDByTitleAndOriginStmt *sql.Stmt
	newArticleStmt                   *sql.Stmt
	newImageStmt                     *sql.Stmt
	updateArticleStatsStmt           *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                               tx,
		tx:                               tx,
		articlesStmt:                     q.articlesStmt,
		attachArticleImageStmt:           q.attachArticleImageStmt,
		getArticleByIDStmt:               q.getArticleByIDStmt,
		getArticleIDByTitleAndOriginStmt: q.getArticleIDByTitleAndOriginStmt,
		newArticleStmt:                   q.newArticleStmt,
		newImageStmt:                     q.newImageStmt,
		updateArticleStatsStmt:           q.updateArticleStatsStmt,
	}
}
