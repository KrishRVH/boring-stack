package appmodel

import "time"

// MaxTodoBodyLength is the maximum number of characters in a todo body.
const MaxTodoBodyLength = 280

// Todo is a todo item rendered by the UI.
type Todo struct {
	ID        string
	Body      string
	Done      bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Event is an application event rendered by the UI.
type Event struct {
	ID        int64
	Kind      string
	Body      string
	CreatedAt time.Time
}

// Stats summarizes todo completion.
type Stats struct {
	Total int64
	Done  int64
}

// Open returns the number of incomplete todos.
func (s Stats) Open() int64 {
	return s.Total - s.Done
}

// VersionInfo lists the application dependency versions shown by the UI.
type VersionInfo struct {
	Go       string
	HTMX     string
	HTMXSSE  string
	Alpine   string
	Templ    string
	Tailwind string
	SQLC     string
	PGX      string
	Goose    string
	River    string
	NATSGo   string
}

// HomeView contains the data rendered on the home page.
type HomeView struct {
	Todos     []Todo
	Events    []Event
	Stats     Stats
	BusName   string
	FormError string
	Version   VersionInfo
}
