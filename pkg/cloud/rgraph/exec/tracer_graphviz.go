/*
Copyright 2023 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exec

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// NewGraphvizTracer returns a new Tracer that outputs Graphviz.
func NewGraphvizTracer() *GraphvizTracer {
	ret := &GraphvizTracer{}
	return ret
}

// GraphvizTracer outputs Graphviz .dot format. This object is thread-safe.
type GraphvizTracer struct {
	lock  sync.Mutex
	start time.Time
	buf   bytes.Buffer
}

var _ Tracer = (*GraphvizTracer)(nil)

func actionTypeToColor(t ActionType) string {
	switch t {
	case ActionTypeCreate:
		return "palegreen"
	case ActionTypeCustom:
		return "khaki"
	case ActionTypeDelete:
		return "pink"
	case ActionTypeMeta:
		return "gray90"
	case ActionTypeUpdate:
		return "khaki1"
	}
	return "magenta"
}

func (tr *GraphvizTracer) outf(s string, args ...any) {
	tr.buf.WriteString(fmt.Sprintf(s+"\n", args...))
}

func (tr *GraphvizTracer) Record(entry *TraceEntry, err error) {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	metadata := entry.Action.Metadata()

	if tr.start.IsZero() {
		tr.start = entry.Start
	}

	tr.outf("  \"%s\" [style=filled,fillcolor=%s,shape=box,label=<", metadata.Name, actionTypeToColor(metadata.Type))
	tr.outf("    <table border=\"0\">")
	tr.outf("      <tr><td colspan=\"2\">\\N</td></tr>")
	tr.outf("      <tr><td colspan=\"2\">%s</td></tr>", metadata.Summary)
	tr.outf("      <tr><td>Start (delta)</td><td>%v</td></tr>", entry.Start.Sub(tr.start))
	tr.outf("      <tr><td>Duration</td><td>%v</td></tr>", entry.End.Sub(entry.Start))
	if err != nil {
		tr.outf("      <tr><td><b>Error</b></td><td><b>%v</b></td></tr>", err)
	}
	tr.outf("    </table>")
	tr.outf("  >]")

	for _, s := range entry.Signaled {
		tr.outf("  \"%s\" -> \"%s\"", entry.Action.Metadata().Name, s.Event)
		tr.outf("  \"%s\" -> \"%s\"", s.Event, s.SignaledAction.Metadata().Name)
	}
}

func (tr *GraphvizTracer) Finish(pending []Action) {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	for _, a := range pending {
		tr.outf("  \"%s\" [style=filled,shape=box,color=pink]\n", a)
		dupe := map[string]struct{}{}
		for _, ev := range a.PendingEvents() {
			if _, ok := dupe[ev.String()]; !ok {
				dupe[ev.String()] = struct{}{}
				tr.outf("  \"%s\" [style=filled,color=pink]\n", ev)
			}
			tr.outf("  \"%s\" -> \"%s\"\n", ev, a)
		}
	}
}

func (tr *GraphvizTracer) String() string {
	tr.lock.Lock()
	defer tr.lock.Unlock()

	var out bytes.Buffer

	out.WriteString("digraph {\n")
	out.WriteString(tr.buf.String())
	out.WriteString("}\n")

	return out.String()
}
