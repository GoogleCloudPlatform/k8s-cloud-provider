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

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	coreexec "os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/graphviz"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/testlib"
	"github.com/kr/pretty"
	"k8s.io/klog/v2"

	_ "github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/testlib/lb"
)

var (
	flags = struct {
		http    string
		dotExec string
	}{
		http:    "localhost:8080",
		dotExec: "/usr/bin/dot",
	}
)

func main() {
	flag.Parse()

	http.HandleFunc("/", mainPage)
	http.HandleFunc("/show", showPage)

	klog.Error(http.ListenAndServe(flags.http, nil))
}

func wrapWriter(w io.Writer) (func(s string), func(f string, args ...any)) {
	outln := func(s string) { w.Write([]byte(s + "\n")) }
	outf := func(f string, args ...any) {
		w.Write([]byte(fmt.Sprintf(f, args...) + "\n"))
	}
	return outln, outf
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	klog.Infof("mainPage")

	outln, outf := wrapWriter(w)

	outln("<!DOCTYPE html>")
	outln("<h1>Test cases</h1>")
	outln("<ol>")

	for _, c := range testlib.Cases() {
		outf("<li><a href=\"/show?q=%s\">%s</a>: %s</li>", c.Name, c.Name, c.Description)
	}
	outln("</ol>")
}

func showPage(w http.ResponseWriter, r *http.Request) {
	klog.Infof("showPage %q", r.URL)

	outln, outf := wrapWriter(w)

	name := r.URL.Query().Get("q")
	tc := testlib.Case(name)
	if tc == nil {
		klog.Infof("showPage %q: 404", name)
		w.WriteHeader(404)
		outf("%q is not a valid test case", name)
		return
	}

	outln("<!DOCTYPE html>")
	outf("<h1>Test case: %s</h1>", name)
	outf("<div>%s</div>", tc.Description)

	cl := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: "proj"})
	for idx, step := range tc.Steps {
		showStep(w, cl, idx, &step)
	}

	outln("<hr />\n")
	outln("<h2>Test case</h2>\n")
	outln("<pre>")
	outln(pretty.Sprint(tc))
	outln("</pre>")

	klog.Infof("showPage %q: 200", name)
}

func showStep(w http.ResponseWriter, cl cloud.Cloud, idx int, step *testlib.Step) {
	outln, outf := wrapWriter(w)

	outln("<hr />")
	outf("<h2>Step %d</h2>", idx)

	if step.SetUp != nil {
		step.SetUp(cl)
	}

	result, err := plan.Do(context.Background(), cl, step.Graph)
	klog.Infof("plan.Do() = _, %v", err)

	outln("<pre>")
	outf("plan.Do() = _, %v", err)
	outln("</pre>")

	if err != nil {
		return
	}

	outln("<h3>Got graph</h3>")
	outln("")
	svg, err := dotSVG(graphviz.Do(result.Got))
	if err == nil {
		outln(svg)
	} else {
		klog.Infof("dotSVG(Got) = _, %v", err)
		outf("dotSVG() = %v", err)
	}
	outln("")

	outln("<h3>Want graph</h3>")
	outln("")
	svg, err = dotSVG(graphviz.Do(result.Want))
	if err == nil {
		outln(svg)
	} else {
		klog.Infof("dotSVG(Want) = _, %v", err)
		outf("dotSVG() = %v", err)
	}
	outln("")

	var viz exec.GraphvizTracer
	ex, err := exec.NewSerialExecutor(cl, result.Actions, exec.DryRunOption(false), exec.TracerOption(&viz))
	if err != nil {
		outf("NewSerialExecutor() = %v, want nil", err)
		return
	}

	execResult, err := ex.Run(context.Background())

	outln("<h3>Plan</h3>")
	outln("")
	svg, err = dotSVG(viz.String())
	if err == nil {
		outln(svg)
	} else {
		klog.Infof("dotSVG(viz) = _, %v", err)
		outf("<pre>dotSVG() = %v</pre>", err)
	}
	outln("")

	outln("<h3>Pending Actions</h3>\n")
	klog.Infof("Pending actions = %d", len(execResult.Pending))

	if len(execResult.Pending) == 0 {
		outln("No pending actions remain; all actions were executable.")
	} else {
		outln("<ol>")
		for _, item := range execResult.Pending {
			outf("<li>%+v</li>", item.Metadata())
		}
		outln("</ol>")
	}
}

func dotSVG(text string) (string, error) {
	klog.Infof("dotSVG: %q", text)
	cmd := coreexec.Command(flags.dotExec, "-Tsvg")

	inPipe, err := cmd.StdinPipe()
	if err != nil {
		klog.Errorf("dotSVG: StdinPipe = %v", err)
		return "", err
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		klog.Errorf("dotSVG: StdoutPipe = %v", err)
		return "", err
	}

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		klog.Errorf("dotSVG: StderrPipe = %v", err)
		return "", err
	}

	var cmdErr error
	cmdDone := make(chan struct{})

	go func() {
		cmdErr = cmd.Run()
		if err != nil {
			klog.Errorf("dotSVG: cmd.Run() = %v", cmdErr)
		}
		close(cmdDone)
	}()

	n, err := inPipe.Write([]byte(text))
	inPipe.Close()

	klog.Infof("dotSVG: Write() = %d, %v", n, err)

	bytes, err := io.ReadAll(outPipe)
	if err != nil {
		klog.Errorf("dotSVG: ReadAll(outPipe) = %v", err)
	}

	errBytes, err := io.ReadAll(errPipe)
	klog.Infof("dotSVG: ReadAll(errPipe) = %v (%q)", err, string(errBytes))

	<-cmdDone
	if cmdErr != nil {
		return "", cmdErr
	}

	// Trim off some of the extra stuff in the output of dot.
	svgText := string(bytes)
	start := strings.Index(svgText, "<svg")
	if start != -1 {
		svgText = svgText[start:]
	}

	return svgText, nil
}
