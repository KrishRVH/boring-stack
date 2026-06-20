package appmodel

import "time"

const MaxTodoBodyLength = 280

type Todo struct {
	ID        string
	Body      string
	Done      bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Event struct {
	ID        int64
	Kind      string
	Body      string
	CreatedAt time.Time
}

type Stats struct {
	Total int64
	Done  int64
}

func (s Stats) Open() int64 {
	return s.Total - s.Done
}

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

type HomeView struct {
	Todos     []Todo
	Events    []Event
	Stats     Stats
	BusName   string
	FormError string
	Version   VersionInfo
}
