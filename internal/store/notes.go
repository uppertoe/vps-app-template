package store

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Note is one note.
type Note struct {
	ID        int64
	Body      string
	CreatedAt time.Time
}

// MaxNoteLength is the longest note, in characters.
const MaxNoteLength = 2000

// ErrNoteLength is returned for an empty or over-long note.
var ErrNoteLength = errors.New("a note must be between 1 and 2000 characters")

// AddNote saves a note.
func (s *Store) AddNote(ctx context.Context, body string) (Note, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > MaxNoteLength {
		return Note{}, ErrNoteLength
	}
	at := now()
	res, err := s.db.ExecContext(ctx, `INSERT INTO notes(body, created_at) VALUES (?, ?)`, body, at)
	if err != nil {
		return Note{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Note{}, err
	}
	t, _ := time.Parse(time.RFC3339Nano, at)
	return Note{ID: id, Body: body, CreatedAt: t}, nil
}

// Notes lists the newest notes first.
func (s *Store) Notes(ctx context.Context, limit int) ([]Note, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, body, created_at FROM notes ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		var at string
		if err := rows.Scan(&n.ID, &n.Body, &at); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountNotes counts every note.
func (s *Store) CountNotes(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM notes`).Scan(&n)
	return n, err
}
