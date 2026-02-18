// SPDX-FileCopyrightText: 2026 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package ui

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/a-h/templ"
	datastar "github.com/starfederation/datastar/sdk/go"
)

//go:generate templ generate

func RenderTempl(w http.ResponseWriter, r *http.Request, cmp templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err := cmp.Render(r.Context(), w)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return err
}

func RenderNewSSE(w http.ResponseWriter, r *http.Request, cmp templ.Component) error {
	sse := datastar.NewSSE(w, r)
	return sse.MergeFragmentTempl(cmp)
}

func RenderSSE(_ http.ResponseWriter, r *http.Request, cmp templ.Component) {
	var buf bytes.Buffer

	writer := io.Writer(&buf)
	_ = cmp.Render(r.Context(), writer)
}
