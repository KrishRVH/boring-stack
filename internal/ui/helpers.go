package ui

import (
	"time"

	"github.com/a-h/templ"
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
