package link

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	dsn := "postgres://shortlink:dev_password@localhost:5432/shortlink_test"

	var err error
	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	schema := `
	CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS links (
	id SERIAL PRIMARY KEY,
	code TEXT UNIQUE NOT NULL,
	url TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
	);
	CREATE TABLE IF NOT EXISTS clicks (
	id SERIAL PRIMARY KEY,
	link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
	clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	ip TEXT,
	user_agent TEXT
	);`
	if _, err := testPool.Exec(ctx, schema); err != nil {
		panic(err)
	}
	code := m.Run()
	testPool.Close()
	os.Exit(code)
}
func cleanDB(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE users, links, clicks RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
}

func insertTestUser(t *testing.T, username string) int {
	t.Helper()
	var id int
	err := testPool.QueryRow(context.Background(),
		"INSERT INTO users (username, password_hash) VALUES ($1, 'fake_hash') RETURNING id",
		username).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func TestCreateAndFindByCode(t *testing.T) {
	cleanDB(t)
	userID := insertTestUser(t, "alice")

	store := NewStore(testPool)
	ctx := context.Background()

	if err := store.Create(ctx, "abc123", "https://example.com", userID); err != nil {
		t.Fatalf("Create: %v", err)
	}
	linkID, url, err := store.FindByCode(ctx, "abc123")
	if err != nil {
		t.Fatalf("FindByCode: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("url: got %q, want %q", url, "https://example.com")
	}
	if linkID <= 0 {
		t.Errorf("linkID: got %d, want > 0", linkID)
	}
}
func TestStore_FindByCode_NotFound(t *testing.T) {
	cleanDB(t)
	store := NewStore(testPool)

	_, _, err := store.FindByCode(context.Background(), "nonexistent")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestStore_ListByUser_FiltersByUser(t *testing.T) {
	cleanDB(t)
	alice := insertTestUser(t, "alice")
	bob := insertTestUser(t, "bob")

	store := NewStore(testPool)
	ctx := context.Background()

	if err := store.Create(ctx, "a1", "https://alice1.com", alice); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "a2", "https://alice2.com", alice); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "b1", "https://bob1.com", bob); err != nil {
		t.Fatal(err)
	}

	aliceLinks, err := store.ListByUser(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceLinks) != 2 {
		t.Errorf("alice should see 2 links, got %d", len(aliceLinks))
	}

	bobLinks, err := store.ListByUser(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobLinks) != 1 {
		t.Errorf("bob should see 1 link, got %d", len(bobLinks))
	}
}
