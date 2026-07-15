package ui

import (
	"time"

	"github.com/a-h/templ"

	"github.com/KrishRVH/boring-stack/internal/appmodel"
)

func attr(key string, value any) templ.Attributes {
	return templ.Attributes{key: value}
}

func attrIf(ok bool, key string, value any) templ.Attributes {
	if !ok {
		return templ.Attributes{}
	}
	return templ.Attributes{key: value}
}

func shortTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("15:04:05")
}

func shortDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("Jan 02 15:04:05")
}

func todoToggleLabel(todo appmodel.Todo) string {
	if todo.Done {
		return "Reopen todo: " + todo.Body
	}
	return "Mark todo done: " + todo.Body
}

func todoDeleteLabel(todo appmodel.Todo) string {
	return "Delete todo: " + todo.Body
}
