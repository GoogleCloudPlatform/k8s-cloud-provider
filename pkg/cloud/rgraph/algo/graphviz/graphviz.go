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

package graphviz

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

// Do returns a .dot (http://graphviz.org) representation of the resource graph
// for visualization.
func Do(g *rgraph.Graph) string {
	var buf bytes.Buffer
	buf.WriteString("digraph G {\n")
	buf.WriteString("  rankdir=TB\n") // layout top to bottom.

	for _, node := range g.All() {
		gn := &viznode{
			name:  node.ID().String(),
			shape: "box",
			style: "filled",
			kv: map[string]any{
				"localPlan": node.Plan().GraphvizString(),
				"state":     node.State(),
			},
		}
		deps := node.OutRefs()
		for _, dep := range deps {
			e := vizedge{from: node.ID(), to: dep.To, field: dep.Path.String()}
			buf.WriteString(e.String())
		}

		gn.fillcolor = gn.opColor(node.Plan().Op())
		buf.WriteString(gn.String())
	}
	buf.WriteString("}\n")

	return buf.String()
}

type viznode struct {
	name string

	fillcolor string
	shape     string
	style     string

	kv map[string]any
}

func (*viznode) indent(n int) string {
	var ret string
	for i := 0; i < n; i++ {
		ret += "  "
	}
	return ret
}

func (*viznode) opColor(op rnode.Operation) string {
	switch op {
	case rnode.OpCreate:
		return "palegreen"
	case rnode.OpDelete:
		return "pink"
	case rnode.OpRecreate:
		return "yellow"
	case rnode.OpUpdate:
		return "khaki1"
	case rnode.OpNothing:
		return "gray90"
	case rnode.OpUnknown:
		return "gray90"
	}
	return "mediumpurple1"
}

func (n *viznode) String() string {
	type line struct {
		indent int
		s      string
	}

	var lines []line

	lines = append(lines, line{1, fmt.Sprintf("\"%s\" [label=<", n.name)})
	lines = append(lines, line{2, "<table border=\"0\">"})
	lines = append(lines, line{3, "<tr><td colspan=\"2\"><font point-size=\"16\">\\N</font></td></tr>"})
	lines = append(lines, line{3, "<tr><td colspan=\"2\">---</td></tr>"})

	var keys []string
	for k := range n.kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, line{3, fmt.Sprintf("<tr><td>%s</td><td align=\"left\">%v</td></tr>", k, n.kv[k])})
	}
	lines = append(lines, line{2, "</table>"})

	var attribsStr string
	for _, at := range []struct {
		key string
		val *string
	}{
		{"fillcolor", &n.fillcolor},
		{"shape", &n.shape},
		{"style", &n.style},
	} {
		if *at.val != "" {
			attribsStr += fmt.Sprintf(`,%s=%s`, at.key, *at.val)
		}
	}
	lines = append(lines, line{1, fmt.Sprintf(">%s]", attribsStr)})

	var out string
	for _, l := range lines {
		out += n.indent(l.indent) + l.s + "\n"
	}
	return out
}

type vizedge struct {
	from, to *cloud.ResourceID
	field    string
}

func (e *vizedge) String() string {
	return fmt.Sprintf("  \"%s\" -> \"%s\" [label=<%s>]\n", e.from, e.to, e.field)
}
