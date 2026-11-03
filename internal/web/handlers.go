package web

import (
	"errors"
	"net/http"

	"github.com/uppertoe/vps-app-template/internal/store"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "home.html", s.site.Title, nil)
}

func (s *Server) privacy(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "privacy.html", "Privacy", nil)
}

type notesView struct {
	Notes     []store.Note
	Error     string
	Body      string // the refused text, shown again
	MaxLength int
}

func (s *Server) notes(w http.ResponseWriter, r *http.Request) {
	notes, err := s.store.Notes(r.Context(), 50)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "notes.html", "Shared notes", notesView{Notes: notes, MaxLength: store.MaxNoteLength})
}

// addNote answers htmx with the refreshed list. A plain form post is
// redirected after a successful save, and shown the page again with the
// error and the typed text when the note is refused, so the page works
// without JavaScript too.
func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	v := notesView{MaxLength: store.MaxNoteLength}
	body := r.FormValue("body")
	if _, err := s.store.AddNote(r.Context(), body); err != nil {
		if !errors.Is(err, store.ErrNoteLength) {
			s.fail(w, r, err)
			return
		}
		v.Error = "A note must be between 1 and 2,000 characters."
		v.Body = body
	}
	htmx := r.Header.Get("HX-Request") == "true"
	if !htmx && v.Error == "" {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
		return
	}
	notes, err := s.store.Notes(r.Context(), 50)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	v.Notes = notes
	if htmx {
		s.renderPartial(w, r, "note-list", v)
		return
	}
	s.render(w, r, http.StatusUnprocessableEntity, "notes.html", "Shared notes", v)
}

type adminView struct {
	NoteCount int
	Group     string
}

func (s *Server) admin(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountNotes(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "admin.html", "Administration", adminView{NoteCount: n, Group: AdminGroup})
}
