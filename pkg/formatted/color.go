// Copyright © 2019 The Tekton Authors.
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

package formatted

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
)

var (
	// Palette of colors for rainbow's tasks, Red is avoided as keeping it for errors
	palette = []color.Attribute{
		color.FgHiGreen,
		color.FgHiYellow,
		color.FgHiBlue,
		color.FgHiMagenta,
		color.FgHiCyan,
	}
)

// DecorateAttr decorate strings with a color or an emoji, respecting the user
// preference if no colour needed.
func DecorateAttr(attrString, message string) string {
	if color.NoColor {
		return message
	}

	switch attrString {
	case "bullet":
		return fmt.Sprintf("∙ %s", message)
	case "check":
		return "✔ ️"
	case "resources":
		return "📦 "
	case "params":
		return "⚓ "
	case "tasks":
		return "🗒  "
	case "pipelineruns":
		return "⛩  "
	case "status":
		return "🌡️  "
	case "inputresources":
		return "📨 "
	case "outputresources":
		return "📡 "
	case "steps":
		return "🦶 "
	case "message":
		return "💌 "
	case "taskruns":
		return "🗂  "
	case "sidecars":
		return "🚗 "
	case "results":
		return "📝 "
	case "workspaces":
		return "📂 "
	case "skippedtasks":
		return "⏭️  "
	case "timeouts":
		return "⏱  "
	}

	attr := color.Reset
	switch attrString {
	case "underline":
		attr = color.Underline
	case "underline bold":
		return color.New(color.Underline).Add(color.Bold).Sprintf(message)
	case "bold":
		attr = color.Bold
	case "yellow":
		attr = color.FgHiYellow
	case "green":
		attr = color.FgHiGreen
	case "red":
		attr = color.FgHiRed
	case "blue":
		attr = color.FgHiBlue
	case "magenta":
		attr = color.FgHiMagenta
	case "cyan":
		attr = color.FgHiCyan
	case "black":
		attr = color.FgHiBlack
	case "white":
		attr = color.FgHiWhite
	}

	return color.New(attr).Sprintf(message)
}

type atomicCounter struct {
	value     uint32
	threshold int
}

func (c *atomicCounter) next() int {
	v := atomic.AddUint32(&c.value, 1)
	next := int(v-1) % c.threshold
	atomic.CompareAndSwapUint32(&c.value, uint32(c.threshold), 0)
	return next

}

type rainbow struct {
	cache   sync.Map
	counter atomicCounter
}

func newRainbow() *rainbow {
	return &rainbow{
		counter: atomicCounter{threshold: len(palette)},
	}
}

func (r *rainbow) get(x string) color.Attribute {
	if value, ok := r.cache.Load(x); ok {
		return value.(color.Attribute)
	}

	clr := palette[r.counter.next()]
	r.cache.Store(x, clr)
	return clr
}

// Fprintf formats according to a format specifier and writes to w.
// the first argument is a label to keep the same colour on.
func (r *rainbow) Fprintf(label string, w io.Writer, format string, args ...interface{}) {
	attribute := r.get(label)
	crainbow := color.Set(attribute).Add(color.Bold)
	crainbow.Fprintf(w, format, args...)
}

// Color formatter to print the colored output on streams
type Color struct {
	Rainbow *rainbow

	red  *color.Color
	blue *color.Color
}

// NewColor returns a new instance color formatter
func NewColor() *Color {
	return &Color{
		Rainbow: newRainbow(),

		red:  color.New(color.FgRed),
		blue: color.New(color.FgBlue),
	}
}

// PrintRed prints the formatted content to given destination in red color
func (c *Color) PrintRed(w io.Writer, format string, args ...interface{}) {
	c.red.Fprintf(w, format, args...)
}

// Error prints the formatted content to given destination in red color
func (c *Color) Error(w io.Writer, format string, args ...interface{}) {
	c.PrintRed(w, format, args...)
}
