// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type SimpleHandler struct {
	Writer io.Writer
	Level  slog.Leveler

	mu sync.Mutex
}

func (h *SimpleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level.Level()
}

func (h *SimpleHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprintf(h.Writer, "%s: %s\n", record.Level.String(), record.Message)
	return err
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// no support for attrs here
	return h
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	// no support for attrs here
	return h
}

var _ slog.Handler = (*SimpleHandler)(nil)
