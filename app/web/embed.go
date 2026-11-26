// Package web embeds the Go html/template files, compiled Tailwind CSS design
// token stylesheet, and pinned vendored HTMX runtime directly into the static
// Tic-Tac-Toe binary for 100% hermetic, zero-CDN, reproducible builds.
package web

import "embed"

// Assets holds templates/*.tmpl and static/* compiled into the binary.
//
//go:embed templates/*.tmpl static/*
var Assets embed.FS
