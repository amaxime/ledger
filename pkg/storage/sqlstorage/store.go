//go:build json1
// +build json1

package sqlstorage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"github.com/numary/ledger/pkg/logging"
	"github.com/pkg/errors"
	"path"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations
var migrations embed.FS

type Store struct {
	flavor  sqlbuilder.Flavor
	ledger  string
	db      *sql.DB
	onClose func(ctx context.Context) error
	logger  logging.Logger
}

func (s *Store) table(name string) string {
	switch Flavor(s.flavor) {
	case PostgreSQL:
		return fmt.Sprintf(`"%s"."%s"`, s.ledger, name)
	default:
		return name
	}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) error(err error) error {
	if err == nil {
		return nil
	}
	return errorFromFlavor(Flavor(s.flavor), err)
}

func NewStore(name string, flavor sqlbuilder.Flavor, db *sql.DB, logger logging.Logger, onClose func(ctx context.Context) error) (*Store, error) {
	return &Store{
		ledger:  name,
		db:      db,
		flavor:  flavor,
		onClose: onClose,
		logger:  logger,
	}, nil
}

func (s *Store) Name() string {
	return s.ledger
}

func (s *Store) Initialize(ctx context.Context) error {
	s.logger.Debug(ctx, "initializing sqlite db")

	statements := make([]string, 0)

	migrationsDir := fmt.Sprintf("migrations/%s", strings.ToLower(s.flavor.String()))

	entries, err := migrations.ReadDir(migrationsDir)

	if err != nil {
		return s.error(err)
	}

	for _, m := range entries {
		s.logger.Debug(ctx, "running migrations %s", m.Name())

		b, err := migrations.ReadFile(path.Join(migrationsDir, m.Name()))
		if err != nil {
			return err
		}

		plain := strings.ReplaceAll(string(b), "VAR_LEDGER_NAME", s.ledger)

		statements = append(
			statements,
			strings.Split(plain, "--statement")...,
		)
	}

	for i, statement := range statements {
		s.logger.Debug(ctx, "running statement: %s", statement)
		_, err = s.db.ExecContext(ctx, statement)
		if err != nil {
			err = errors.Wrapf(s.error(err), "failed to run statement %d", i)
			s.logger.Error(ctx, "%s", err)
			return err
		}
	}

	return nil
}

func (s *Store) Close(ctx context.Context) error {
	err := s.onClose(ctx)
	if err != nil {
		return err
	}
	return nil
}
