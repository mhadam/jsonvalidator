package main

import (
	"database/sql"
)

type jsonSchema struct {
	SchemaID  string  `json:"schemaId"`
	SchemaDef string `json:"schemaDef"`
}

func (s *jsonSchema) getSchema(db *sql.DB) error {
	err := db.QueryRow("SELECT json_schema_id, json_schema_def FROM json_schema WHERE json_schema_id=$1",
		s.SchemaID).Scan(&s.SchemaID, &s.SchemaDef)

	return err
}

func (s *jsonSchema) updateSchema(db *sql.DB) error {
	_, err :=
		db.Exec("UPDATE json_schema SET json_schema_id=$1, json_schema_def=$2 WHERE id=$2",
			s.SchemaID, s.SchemaDef)

	return err
}

func (s *jsonSchema) createSchema(db *sql.DB) error {
	err := db.QueryRow(
		"INSERT INTO json_schema(json_schema_id, json_schema_def) VALUES($1, $2) RETURNING json_schema_id, json_schema_def",
		s.SchemaID, s.SchemaDef).Scan(&s.SchemaID, &s.SchemaDef)

	if err != nil {
		return err
	}

	return nil
}