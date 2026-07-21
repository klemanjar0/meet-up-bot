package telegram

import (
	"sync"
	"time"

	"meet-up-bot/internal/storage/db"
)

// step identifies where a user is in the lobby-creation conversation.
type step int

const (
	stepIdle step = iota
	stepName
	stepDescription
	stepCountry
	stepCity
	stepAddress
	stepTime
	stepChatLink
	stepVisibility
)

// editField names a single lobby field being edited via the /mylobbies → Edit
// flow (visibility is toggled directly by a button, so it isn't listed here).
type editField string

const (
	editFieldName        editField = "name"
	editFieldDescription editField = "description"
	editFieldCountry     editField = "country"
	editFieldCity        editField = "city"
	editFieldAddress     editField = "address"
	editFieldTime        editField = "time"
	editFieldLink        editField = "link"
)

// settingsField names a user setting collected via free-text input.
type settingsField string

const (
	settingsFieldTimezone settingsField = "timezone"
	settingsFieldCity     settingsField = "city"
)

// sessionKind distinguishes the conversation types a user can be in.
type sessionKind int

const (
	kindCreate sessionKind = iota
	kindEdit
	kindSettings
)

// session holds the in-progress conversation for a user. A create session walks
// through every field via step; an edit session collects a single new value for
// editField of lobby editLobbyID.
type session struct {
	kind sessionKind

	// create wizard
	step        step
	name        string
	description string
	country     string
	city        string
	address     string
	eventTime   time.Time
	chatLink    string
	visibility  db.LobbyVisibility

	// edit flow
	editLobbyID int64
	editField   editField

	// settings flow
	settingsField settingsField
}

// sessions is a concurrency-safe store of per-user conversations, kept in
// memory: an interrupted session is simply lost, which is fine for a short
// wizard.
type sessions struct {
	mu sync.Mutex
	m  map[int64]*session
}

func newSessions() *sessions {
	return &sessions{m: make(map[int64]*session)}
}

// startCreate begins a fresh creation wizard for the user.
func (s *sessions) startCreate(userID int64) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := &session{kind: kindCreate, step: stepName}
	s.m[userID] = sess
	return sess
}

// startEdit begins collecting a new value for a single lobby field.
func (s *sessions) startEdit(userID, lobbyID int64, field editField) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := &session{kind: kindEdit, editLobbyID: lobbyID, editField: field}
	s.m[userID] = sess
	return sess
}

// startSettings begins collecting a free-text value for a user setting.
func (s *sessions) startSettings(userID int64, field settingsField) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := &session{kind: kindSettings, settingsField: field}
	s.m[userID] = sess
	return sess
}

// get returns the user's active session, or nil if none is in progress.
func (s *sessions) get(userID int64) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[userID]
}

// clear removes the user's session.
func (s *sessions) clear(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, userID)
}
