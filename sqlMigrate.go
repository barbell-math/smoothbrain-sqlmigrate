// A very simple library that provides a way to perform SQL migrations using
// pgxDB.
package sbsqlm

//go:generate sqlc generate

import (
	"context"
	"embed"
	"errors"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"

	sberr "github.com/barbell-math/smoothbrain-errs"
	"github.com/barbell-math/smoothbrain-sqlmigrate/queries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	migration int

	// The function signature for any golang operations that should be performed
	// after the sql migration has been executed.
	PostMigrationOp func(ctxt context.Context, p *pgxpool.Pool) error

	Migrations struct {
		sqlMigrations  map[migration]string
		goMigrations   map[migration]PostMigrationOp
		embedFs        embed.FS
		dir            string
		maxMigrationId migration
	}
)

var (
	// The global migration struct for the package that is used by the bare
	// [Load], [Status], and [Run] functions.
	migrations Migrations

	MalformedMigrationFileErr = errors.New("Malformed migration file")
	MigrationSequenceErr      = errors.New("Malformed migration sequence")
	MissingSqlMigrationErr    = errors.New(
		"A postOp did not have an associated sql migration file",
	)
	UnknownMigrationErr = errors.New(
		"A migration requested from the database was not found",
	)
)

// Loads a set of migration files from the supplied directory of the supplied
// embedded file system. All sql migration files are expected to have an integer
// for the file name and end with a sql file extension. Migration files are
// expected to be strictly increasing with no missing numbers.
//
// postOps define code that will be run after the corresponding sql migration
// has been run, as defined by the sql file number. Post ops are not required.
//
// An embedded file system is used to encourage all migrations to be baked into
// the executable so that deploying any application will not also require
// deploying a dir containing all the migration files.
func Load(fs embed.FS, dir string, postOps map[migration]PostMigrationOp) error {
	return migrations.Load(fs, dir, postOps)
}

// Returns the current status of all migrations. This result will include the
// status of all operations that are not yet listed in the database.
func Status(ctxt context.Context, p *pgxpool.Pool) (
	[]queries.SmoothbrainSqlmigrateVersioning,
	error,
) {
	return migrations.Status(ctxt, p)
}

// Runs all migrations that need to be run. This will run all migrations that
// have a status of false in the database and will run any additional migrations
// that have been added. All migrations will be run in the increasing order and
// if an error is encountered all further migrations will not be run.
//
// All migrations will be run inside a transaction. If any migration fails the
// entire transaction will be rolled back, there will be no changes to the
// database, and any migrations that ran successfully before the failed
// migration will need to be re-run.
func Run(ctxt context.Context, p *pgxpool.Pool) error {
	return migrations.Run(ctxt, p)
}

// Loads a set of migration files from the supplied directory of the supplied
// embedded file system. All sql migration files are expected to have an integer
// for the file name and end with a sql file extension. Migration files are
// expected to be strictly increasing with no missing numbers.
//
// postOps define code that will be run after the corresponding sql migration
// has been run, as defined by the sql file number. Post ops are not required.
//
// An embedded file system is used to encourage all migrations to be baked into
// the executable so that deploying any application will not also require
// deploying a dir containing all the migration files.
func (m *Migrations) Load(
	sqlFiles embed.FS,
	dir string,
	postOps map[migration]PostMigrationOp,
) error {
	m.sqlMigrations = map[migration]string{}
	m.goMigrations = postOps
	m.embedFs = sqlFiles
	m.dir = dir
	m.maxMigrationId = 0

	files, err := sqlFiles.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		fName := f.Name()
		if !strings.HasSuffix(fName, ".sql") {
			continue
		}

		iterMigration, err := strconv.ParseInt(fName[:len(fName)-4], 10, 0)
		if err != nil {
			return sberr.AppendError(MalformedMigrationFileErr, err)
		}
		m.sqlMigrations[migration(iterMigration)] = fName

		if iterMigration > int64(m.maxMigrationId) {
			m.maxMigrationId = migration(iterMigration)
		}
	}

	allIds := slices.Collect(maps.Keys(m.sqlMigrations))
	slices.Sort(allIds)
	for i, id := range allIds {
		if migration(i) != id {
			return sberr.Wrap(
				MigrationSequenceErr,
				"Missing %d migration. Migrations must be sequential with no gaps.",
				i,
			)
		}
	}

	for mId := range m.goMigrations {
		if _, ok := m.sqlMigrations[mId]; !ok {
			return sberr.Wrap(MissingSqlMigrationErr, "Missing ID: %d", mId)
		}
	}

	return nil
}

// Returns the current status of all migrations. This result will include the
// status of all operations that are not yet listed in the database.
func (m *Migrations) Status(
	ctxt context.Context,
	p *pgxpool.Pool,
) ([]queries.SmoothbrainSqlmigrateVersioning, error) {
	q := queries.New(p)
	rv := []queries.SmoothbrainSqlmigrateVersioning{}

	tableExists, err := q.VersioningExists(ctxt)
	if err != nil {
		return rv, err
	} else if !tableExists {
		return rv, nil
	}

	rv, err = q.Status(ctxt)
	maxIdInDB, err := q.MaxID(ctxt)
	if err != nil {
		return rv, err
	}

	for i := maxIdInDB.(int32) + 1; i <= int32(m.maxMigrationId); i++ {
		rv = append(rv, queries.SmoothbrainSqlmigrateVersioning{
			ID: i,
			Ok: false,
		})
	}
	slices.SortFunc(rv, func(a, b queries.SmoothbrainSqlmigrateVersioning) int {
		return int(a.ID - b.ID)
	})

	return rv, nil
}

// Runs all migrations that need to be run. This will run all migrations that
// have a status of false in the database and will run any additional migrations
// that have been added. All migrations will be run in the increasing order and
// if an error is encountered all further migrations will not be run.
//
// All migrations will be run inside a transaction. If any migration fails the
// entire transaction will be rolled back, there will be no changes to the
// database, and any migrations that ran successfully before the failed
// migration will need to be re-run.
func (m *Migrations) Run(ctxt context.Context, p *pgxpool.Pool) error {
	q := queries.New(p)
	tx, err := p.BeginTx(ctxt, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctxt)
	qtx := q.WithTx(tx)

	if err := qtx.CreateSchema(ctxt); err != nil {
		return err
	}
	if err := qtx.CreateVersioning(ctxt); err != nil {
		return err
	}

	needToRun, err := qtx.NeedToBeRun(ctxt)
	if err != nil {
		return err
	}
	maxIdInDB, err := qtx.MaxID(ctxt)
	if err != nil {
		return err
	}

	for i := maxIdInDB.(int32) + 1; i <= int32(m.maxMigrationId); i++ {
		needToRun = append(needToRun, i)
	}
	slices.Sort(needToRun)

	var sql []byte
	for _, id := range needToRun {
		fName, ok := m.sqlMigrations[migration(id)]
		if !ok {
			return sberr.Wrap(UnknownMigrationErr, "Migration ID: %d", id)
		}

		if sql, err = m.embedFs.ReadFile(path.Join(m.dir, fName)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctxt, string(sql)); err != nil {
			return err
		}
		if op, ok := m.goMigrations[migration(id)]; ok {
			if err = op(ctxt, p); err != nil {
				return err
			}
		}

		if err = qtx.SetStatus(ctxt, queries.SetStatusParams{
			Ok: true, ID: id,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctxt)
}
