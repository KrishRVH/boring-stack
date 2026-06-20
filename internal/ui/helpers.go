package ui

import (
	"fmt"
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
		return fmt.Sprintf("Reopen todo: %s", todo.Body)
	}
	return fmt.Sprintf("Mark todo done: %s", todo.Body)
}

func todoDeleteLabel(todo appmodel.Todo) string {
	return fmt.Sprintf("Delete todo: %s", todo.Body)
}
