//go:build integration

package migrations_test

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/apotourlyan/ludustudii/migrations"
	"github.com/pressly/goose/v3"
)

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m))
}

// schemaObjects lists everything in the public schema other than goose's own
// bookkeeping (goose_db_version, its sequence, pkey index and constraints),
// sorted so two snapshots can be compared directly. Each entry starts with its
// kind and includes the object's definition where Postgres can render one, so
// a Down that leaves an object redefined under the same name still shows up as
// a difference.
func schemaObjects(t *testing.T, db *sql.DB) []string {
	t.Helper()
	objects := slices.Concat(
		relations(t, db),
		types(t, db),
		functions(t, db),
		columns(t, db),
		constraints(t, db),
		triggers(t, db),
	)
	slices.Sort(objects)
	return objects
}

// relations lists tables, sequences, indexes, views and materialized views
// from pg_class. Indexes carry pg_get_indexdef (columns, method, uniqueness,
// predicate), and views carry pg_get_viewdef (the SELECT they expand to).
// Tables are listed by name only; their shape comes from columns and
// constraints.
func relations(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "relations", `
		SELECT 'relation ' || c.relname || CASE
			WHEN c.relkind = 'i' THEN ' ' || pg_get_indexdef(c.oid)
			WHEN c.relkind IN ('v', 'm') THEN ' ' || pg_get_viewdef(c.oid)
			ELSE ''
		END
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname NOT LIKE 'goose\_db\_version%'`)
}

// types lists user-defined enums ('e'), domains ('d'), ranges ('r') and
// multiranges ('m'). Enums carry their labels in sort order, so an ADD VALUE
// that is not reversed shows up. The coalesce keeps an enum with no labels from
// turning the row into NULL. Domain CHECKs are covered by constraints.
func types(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "types", `
		SELECT 'type ' || t.typname || CASE WHEN t.typtype = 'e' THEN ' (' || coalesce(
			(SELECT string_agg(e.enumlabel, ', ' ORDER BY e.enumsortorder)
			 FROM pg_enum e WHERE e.enumtypid = t.oid), '') || ')' ELSE '' END
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = 'public' AND t.typtype IN ('e', 'd', 'r', 'm')`)
}

// functions lists functions and procedures by full signature, so overloads
// are told apart. Plain functions ('f'), procedures ('p') and window functions
// ('w') carry pg_get_functiondef; aggregates ('a') are listed by signature only
// because pg_get_functiondef raises an error for them.
func functions(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "functions", `
		SELECT 'function ' || p.oid::regprocedure::text || CASE
			WHEN p.prokind IN ('f', 'p', 'w') THEN ' ' || pg_get_functiondef(p.oid)
			ELSE ''
		END
		FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public'`)
}

// columns lists the columns of ordinary ('r') and partitioned ('p') tables and
// the fields of composite types ('c') as owner.column with the exact type from
// format_type (including length or precision, such as character varying(200)),
// NOT NULL and the default expression. attnum > 0 skips system columns like
// ctid, and NOT attisdropped skips the placeholder Postgres keeps for a dropped
// column.
func columns(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "columns", `
		SELECT 'column ' || c.relname || '.' || a.attname || ' ' || format_type(a.atttypid, a.atttypmod)
			|| CASE WHEN a.attnotnull THEN ' not null' ELSE '' END
			|| coalesce(' default ' || pg_get_expr(d.adbin, d.adrelid), '')
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p', 'c')
			AND a.attnum > 0 AND NOT a.attisdropped
			AND c.relname NOT LIKE 'goose\_db\_version%'`)
}

// constraints lists table and domain constraints (primary keys, foreign keys,
// unique, CHECK, exclusion and, since Postgres 18, NOT NULL) with their
// pg_get_constraintdef definition, so a CHECK that is dropped and re-added
// under the same name with a different expression shows up. Constraint names
// are unique only per table, so each line is prefixed with its owner: the table,
// or the type for a domain constraint. A CASE picks the owner because a domain
// constraint has conrelid = 0, which casts to regclass as '-' rather than NULL.
func constraints(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "constraints", `
		SELECT 'constraint '
			|| CASE WHEN co.conrelid <> 0 THEN co.conrelid::regclass::text
			        ELSE co.contypid::regtype::text END
			|| '.' || co.conname || ' ' || pg_get_constraintdef(co.oid)
		FROM pg_constraint co JOIN pg_namespace n ON n.oid = co.connamespace
		WHERE n.nspname = 'public' AND co.conname NOT LIKE 'goose\_db\_version%'`)
}

// triggers lists user-defined triggers with their pg_get_triggerdef
// definition. A trigger is not caught by functions when its function comes
// from an earlier migration, so it needs its own entry. NOT tgisinternal skips
// the triggers Postgres creates to enforce foreign keys.
func triggers(t *testing.T, db *sql.DB) []string {
	t.Helper()
	return queryOutput(t, db, "triggers", `
		SELECT 'trigger ' || tg.tgname || ' ' || pg_get_triggerdef(tg.oid)
		FROM pg_trigger tg
		JOIN pg_class c ON c.oid = tg.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND NOT tg.tgisinternal`)
}

// queryOutput runs a query that returns a single text column and collects the
// rows. kind names the query in failure messages.
func queryOutput(t *testing.T, db *sql.DB, kind, query string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("query %q: %v", kind, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan %q: %v", kind, err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %q: %v", kind, err)
	}
	return out
}

// subtract returns the items of a that are not in b.
func subtract(a, b []string) []string {
	var out []string
	for _, s := range a {
		if !slices.Contains(b, s) {
			out = append(out, s)
		}
	}
	return out
}

func TestMigrations_UpDownUp(t *testing.T) {
	t.Parallel()

	db := testkit.NewDB(t)
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	// Full down from the migrated template must leave an empty schema.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	_, err = p.DownTo(ctx, 0)
	cancel()
	if err != nil {
		t.Fatalf("down to 0: %v", err)
	}
	if left := schemaObjects(t, db); len(left) > 0 {
		t.Fatalf("leftover objects after down to 0: got %q, want none", left)
	}

	// Each Down runs with all earlier migrations applied and must restore the
	// schema exactly as it was before its Up.
	sources := p.ListSources()
	if len(sources) == 0 {
		t.Fatal("no migration sources")
	}
	for _, src := range sources {
		before := schemaObjects(t, db)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		_, err := p.UpByOne(ctx)
		cancel()
		if err != nil {
			t.Fatalf("version %d: up: %v", src.Version, err)
		}

		ctx, cancel = context.WithTimeout(t.Context(), 10*time.Second)
		_, err = p.Down(ctx)
		cancel()
		if err != nil {
			t.Fatalf("version %d: down: %v", src.Version, err)
		}
		if after := schemaObjects(t, db); !slices.Equal(before, after) {
			t.Fatalf("version %d: down did not restore schema:\n missing %q\n extra   %q",
				src.Version, subtract(before, after), subtract(after, before))
		}

		ctx, cancel = context.WithTimeout(t.Context(), 10*time.Second)
		_, err = p.UpByOne(ctx)
		cancel()
		if err != nil {
			t.Fatalf("version %d: up again: %v", src.Version, err)
		}
	}

	ctx, cancel = context.WithTimeout(t.Context(), 10*time.Second)
	got, err := p.GetDBVersion(ctx)
	cancel()
	if err != nil {
		t.Fatalf("get db version: %v", err)
	}
	if want := sources[len(sources)-1].Version; got != want {
		t.Errorf("db version: got %v, want %v", got, want)
	}

	ctx, cancel = context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var exists bool
	err = db.QueryRowContext(ctx,
		`SELECT to_regclass('public.learning_cards') IS NOT NULL`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("query learning_cards: %v", err)
	}
	if !exists {
		t.Errorf("learning_cards exists: got %v, want %v", exists, true)
	}
}
