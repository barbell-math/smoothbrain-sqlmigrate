-- name: CreateSchema :exec
CREATE SCHEMA IF NOT EXISTS smoothbrain_sqlmigrate;

-- name: CreateVersioning :exec
CREATE TABLE IF NOT EXISTS smoothbrain_sqlmigrate.versioning (
	id INT NOT NULL UNIQUE,
	ok BOOLEAN NOT NULL
);

-- name: VersioningExists :one
SELECT EXISTS (
   SELECT FROM information_schema.tables 
   WHERE  table_schema = 'smoothbrain_sqlmigrate'
   AND    table_name   = 'versioning'
);

-- name: NeedUpdate :one
SELECT EXISTS (
	SELECT 1
	FROM   smoothbrain_sqlmigrate.versioning
	WHERE  ok = false
	LIMIT  1
);

-- name: Status :many
SELECT * FROM smoothbrain_sqlmigrate.versioning ORDER BY id ASC;

-- name: NeedToBeRun :many
SELECT id FROM smoothbrain_sqlmigrate.versioning WHERE ok=false ORDER BY id ASC;

-- name: SetStatus :exec
INSERT INTO smoothbrain_sqlmigrate.versioning (
	id, ok
) VALUES ($1, $2) ON CONFLICT(id) DO UPDATE SET ok=$2;

-- name: MaxID :one
SELECT COALESCE(MAX(id), -1) FROM smoothbrain_sqlmigrate.versioning;
