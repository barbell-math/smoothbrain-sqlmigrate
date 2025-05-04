CREATE SCHEMA IF NOT EXISTS smoothbrain_sqlmigrate;

CREATE TABLE IF NOT EXISTS smoothbrain_sqlmigrate.versioning (
	id INT NOT NULL UNIQUE,
	ok BOOLEAN NOT NULL
);
