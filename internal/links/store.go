package link

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

type Link struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) Create(ctx context.Context, code, url string, userID int) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO links (code, url, user_id) VALUES ($1, $2, $3)",
		code, url, userID)
	return err
}
func (s *Store) FindByCode(ctx context.Context, code string) (int, string, error) {
	var id int
	var url string
	err := s.db.QueryRow(ctx,
		"SELECT id, url FROM links WHERE code = $1", code,
	).Scan(&id, &url)
	return id, url, err
}
func (s *Store) ListByUser(ctx context.Context, userID int) ([]Link, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, url, created_at
		FROM links
		WHERE user_id = $1
		ORDER BY created_at DESC
		`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Link{}
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.Code, &l.URL, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}
func (s *Store) RecordClick(ctx context.Context, linkID int, ip, userAgent string) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO clicks (link_id, ip, user_agent) VALUES ($1, $2, $3)",
		linkID, ip, userAgent,
	)
	return err
}
