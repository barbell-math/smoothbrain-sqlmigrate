package sbsqlm

import (
	"context"
	"embed"
	"testing"

	sbtest "github.com/barbell-math/smoothbrain-test"
	"github.com/jackc/pgx/v5"
)

var (
	//go:embed testData/ok/*
	okFs embed.FS

	//go:embed testData/badFileName/*
	badFileName embed.FS

	//go:embed testData/badSequence/*
	badSequence embed.FS
)

func TestLoadFilesBadFileName(t *testing.T) {
	m := Migrations{}
	err := m.Load(
		badFileName,
		"testData/badFileName",
		map[Migration]PostMigrationOp{},
	)
	sbtest.ContainsError(t, MalformedMigrationFileErr, err)
}

func TestLoadFilesBadSequence(t *testing.T) {
	m := Migrations{}
	err := m.Load(
		badSequence,
		"testData/badSequence",
		map[Migration]PostMigrationOp{},
	)
	sbtest.ContainsError(t, MigrationSequenceErr, err)
}

func TestLoadFilesMissingPostOp(t *testing.T) {
	m := Migrations{}
	err := m.Load(okFs, "testData/ok", map[Migration]PostMigrationOp{
		4: func(ctxt context.Context, tx pgx.Tx) error { return nil },
	})
	sbtest.ContainsError(t, MissingSqlMigrationErr, err)
}

func TestLoadFilesOk(t *testing.T) {
	m := Migrations{}
	err := m.Load(okFs, "testData/ok", map[Migration]PostMigrationOp{})
	sbtest.Nil(t, err)
	sbtest.MapsMatch(t, m.sqlMigrations, map[Migration]string{
		0: "0.sql",
		1: "1.sql",
		2: "2.sql",
		3: "3.sql",
	})
}

// Used for testing with a db
// func TestTemp(t *testing.T) {
// 	ctx := context.Background()
//
// 	conn, err := pgxpool.New(ctx, "user=jack dbname=migratetest sslmode=verify-full")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer conn.Close()
// 	fmt.Println(Load(okFs, "testData/ok", map[migration]PostMigrationOp{}))
// 	fmt.Println(Run(ctx, conn))
// 	fmt.Println(Status(ctx, conn))
// }
