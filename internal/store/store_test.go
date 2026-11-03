package store

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNotes(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if _, err := s.AddNote(ctx, "  first  "); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddNote(ctx, "second"); err != nil {
		t.Fatal(err)
	}
	notes, err := s.Notes(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 || notes[0].Body != "second" || notes[1].Body != "first" {
		t.Fatalf("%+v", notes)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("é", MaxNoteLength+1)} {
		if _, err := s.AddNote(ctx, bad); !errors.Is(err, ErrNoteLength) {
			t.Errorf("accepted a note of %d characters", len([]rune(bad)))
		}
	}
	// Text that looks like SQL is stored as text.
	if _, err := s.AddNote(ctx, "'); DROP TABLE notes; --"); err != nil {
		t.Fatal(err)
	}
	if n, err := s.CountNotes(ctx); err != nil || n != 3 {
		t.Fatalf("count %d %v", n, err)
	}
}
