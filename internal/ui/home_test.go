package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/KrishRVH/boring-stack/internal/appmodel"
)

func TestHomeUsesHTMXSSEAndAlpineAssets(t *testing.T) {
	html := render(t, Home(appmodel.HomeView{
		BusName: "memory",
		Stats:   appmodel.Stats{Total: 1},
		Version: appmodel.VersionInfo{Go: "go-test"},
	}))

	for _, want := range []string{
		`/assets/vendor/htmx.min.js`,
		`/assets/vendor/sse.min.js`,
		`/assets/vendor/alpine.min.js`,
		`hx-ext="sse"`,
		`sse-connect="/stream"`,
		`sse-swap="snapshot"`,
		`hx-swap="none"`,
		`maxlength="280"`,
		`x-data=`,
		`x-text=`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered home missing %q", want)
		}
	}

	for _, stale := range []string{"datastar", "data-on", "data-bind", "data-signals"} {
		if strings.Contains(strings.ToLower(html), stale) {
			t.Fatalf("rendered home still contains %q", stale)
		}
	}
}

func TestSnapshotPanelsRenderOOBSwaps(t *testing.T) {
	html := render(
		t,
		StatsPanel(appmodel.Stats{Total: 2, Done: 1}, true),
		TodosPanel([]appmodel.Todo{{ID: "todo-1", Body: "ship it"}}, true),
		EventsPanel([]appmodel.Event{{Kind: "todo", Body: "created"}}, true),
	)

	for _, want := range []string{
		`id="stats-panel"`,
		`id="todos-panel"`,
		`id="events-panel"`,
		`hx-swap-oob="outerHTML"`,
		`hx-post="/todos/todo-1/toggle"`,
		`hx-delete="/todos/todo-1"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("snapshot fragments missing %q", want)
		}
	}
}

func render(t *testing.T, components ...templ.Component) string {
	t.Helper()

	var b bytes.Buffer
	for _, component := range components {
		if err := component.Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
	}
	return b.String()
}
