package diagnostics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"math"
	"strings"
)

type Report struct {
	Graph   Graph           `json:"graph"`
	Modules ModuleGraph     `json:"modules"`
	Startup Startup         `json:"startup"`
	Cold    *ProcessStartup `json:"coldStartup,omitempty"`
	Scale   []ScaleStartup  `json:"scale,omitempty"`
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
func graphSVG(g Graph) template.HTML {
	var b strings.Builder
	n := len(g.Nodes)
	fmt.Fprintf(&b, "<svg viewBox=\"0 0 1000 650\" role=\"img\" aria-label=\"%d providers, %d directed dependency edges\"><defs><marker id=\"arrow\" markerWidth=\"8\" markerHeight=\"8\" refX=\"7\" refY=\"4\" orient=\"auto\"><path d=\"M0 0 L8 4 L0 8 Z\" fill=\"#94a3b8\"/></marker></defs>", n, len(g.Edges))
	pos := map[string][2]float64{}
	for i, node := range g.Nodes {
		angle := 2*math.Pi*float64(i)/float64(max(n, 1)) - math.Pi/2
		pos[node.ID] = [2]float64{500 + 300*math.Cos(angle), 325 + 205*math.Sin(angle)}
	}
	for _, edge := range g.Edges {
		a, bp := pos[edge.From], pos[edge.To]
		fmt.Fprintf(&b, "<line x1=\"%.1f\" y1=\"%.1f\" x2=\"%.1f\" y2=\"%.1f\" stroke=\"#64748b\" stroke-width=\"1.8\" marker-end=\"url(#arrow)\"/>", a[0], a[1], bp[0], bp[1])
	}
	cyclic := map[string]bool{}
	for _, group := range g.Cycles {
		for _, name := range group {
			cyclic[name] = true
		}
	}
	for _, node := range g.Nodes {
		p := pos[node.ID]
		stroke := "#52677e"
		if cyclic[node.ID] {
			stroke = "#fb7185"
		}
		label := []rune(node.ID)
		if len(label) > 36 {
			label = append([]rune("…"), label[len(label)-35:]...)
		}
		fmt.Fprintf(&b, "<rect x=\"%.1f\" y=\"%.1f\" width=\"250\" height=\"44\" rx=\"10\" fill=\"#202d40\" stroke=\"%s\"/><text x=\"%.1f\" y=\"%.1f\" text-anchor=\"middle\" font-size=\"12\" fill=\"white\" font-family=\"monospace\">%s</text>", p[0]-125, p[1]-22, stroke, p[0], p[1]+4, html.EscapeString(string(label)))
	}
	if n == 0 {
		b.WriteString("<text x=\"500\" y=\"325\" text-anchor=\"middle\" fill=\"#94a3b8\">No providers</text>")
	}
	b.WriteString("</svg>")
	return template.HTML(b.String())
}
func startupSVG(s Startup) template.HTML {
	if len(s.SamplesNS) == 0 {
		return template.HTML("<p>No measurements.</p>")
	}
	var b strings.Builder
	b.WriteString("<svg viewBox=\"0 0 1000 245\" role=\"img\" aria-label=\"Measured nanoseconds per runtime.New call by sample\">")
	maxVal := s.P95NS
	for _, v := range s.SamplesNS {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}
	for i, v := range s.SamplesNS {
		x := 65 + float64(i)*860/float64(len(s.SamplesNS))
		w := math.Min(45, 860/float64(len(s.SamplesNS))*.8)
		h := math.Max(1, 150*v/maxVal)
		fmt.Fprintf(&b, "<rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" rx=\"3\" fill=\"#e11d48\"><title>Sample %d: %.0f ns/op</title></rect>", x, 192-h, w, h, i+1, v)
	}
	y := 192 - 150*s.P95NS/maxVal
	fmt.Fprintf(&b, "<line x1=\"55\" y1=\"%.1f\" x2=\"950\" y2=\"%.1f\" stroke=\"#fbbf24\" stroke-width=\"2\" stroke-dasharray=\"6 4\"/><text x=\"65\" y=\"228\" fill=\"#fbbf24\" font-size=\"13\">p95: %.0f ns/op</text>", y, y, s.P95NS)
	b.WriteString("</svg>")
	return template.HTML(b.String())
}

var htmlTemplate = template.Must(template.New("diagnostics").Funcs(template.FuncMap{"join": strings.Join, "micro": func(ns float64) float64 { return ns / 1000 }}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>NestGo diagnostic report</title>
<style>
:root{color-scheme:dark;background:#0b1220;color:#f8fafc;font:15px/1.5 system-ui,sans-serif}*{box-sizing:border-box}body{margin:0}main{max-width:1180px;padding:35px 22px 70px;margin:auto}
h1{font-size:clamp(28px,5vw,46px);letter-spacing:-.04em;margin:4px 0}h2{font-size:20px;margin-top:0}
header{padding-bottom:20px;border-bottom:1px solid #334155}.eyebrow{color:#fb7185;font-weight:700;letter-spacing:.18em;font-size:12px}
.panel{border:1px solid #334155;border-radius:14px;background:#151f30;padding:22px;margin-top:20px;overflow:auto}
.metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px;margin:20px 0}
.metric{background:#202d40;border-radius:10px;padding:18px}.metric strong{display:block;font-size:24px}.metric span,.muted{color:#a3b3c7;font-size:13px}
svg{width:100%;max-height:740px}table{width:100%;border-collapse:collapse}td,th{text-align:left;border-bottom:1px solid #334155;padding:9px;font-size:13px}
th{color:#94a3b8}code{color:#fda4af;overflow-wrap:anywhere}.warning{border-left:3px solid #fb7185;padding:6px 10px}
footer{margin-top:22px;color:#94a3b8;font-size:13px}
</style></head><body><main>
<header><div class="eyebrow">NESTGO / DIAGNOSTICS</div><h1>Dependency graph &amp; startup</h1><p class="muted">Self-contained performance report · {{.Startup.Scenario}}</p></header>
<div class="metrics">
<div class="metric"><strong>{{len .Graph.Nodes}}</strong><span>Providers</span></div>
<div class="metric"><strong>{{len .Graph.Edges}}</strong><span>Dependency edges</span></div>
<div class="metric"><strong>{{len .Graph.Cycles}}</strong><span>Cycle components</span></div>
<div class="metric"><strong>{{len .Graph.Findings}}</strong><span>Injection findings</span></div>
<div class="metric"><strong>{{printf "%.2f" .MedianUS}} µs</strong><span>Startup median</span></div>
<div class="metric"><strong>{{printf "%.2f" .P95US}} µs</strong><span>Startup p95</span></div>
<div class="metric"><strong>{{.Startup.BytesPerOp}} B</strong><span>Bytes per startup</span></div>
<div class="metric"><strong>{{.Startup.AllocsPerOp}}</strong><span>Allocs per startup</span></div>
</div>
<section class="panel"><h2>Provider dependency graph</h2><p class="muted">Red-outline nodes belong to cycles. Structural output is stable for identical inputs.</p>{{.GraphSVG}}</section>
<section class="panel"><h2>Measured startup performance</h2><p class="muted">Actual wall-clock samples per runtime.New invocation. Excludes process boot, compilation, package init and socket listening.</p>{{.StartupSVG}}
<p class="muted">{{len .Startup.SamplesNS}} samples × {{.Startup.IterationsPerSample}} calls; {{.Startup.GoVersion}} · {{.Startup.GOOS}}/{{.Startup.GOARCH}} · {{.Startup.CPUCount}} CPUs · commit {{.Startup.Commit}}</p></section>
{{if .Cold}}<section class="panel"><h2>Cold process startup</h2><p class="muted">Real launch-to-exit time from an already-built executable; includes OS process boot, Go package initialization and NestGo runtime.New. It excludes compilation and network listening.</p><div class="metrics"><div class="metric"><strong>{{printf "%.2f" .ColdMedianMS}} ms</strong><span>Cold median</span></div><div class="metric"><strong>{{printf "%.2f" .ColdP95MS}} ms</strong><span>Cold p95</span></div><div class="metric"><strong>{{len .Cold.SamplesNS}}</strong><span>Process launches</span></div></div><p class="muted">Commit {{.Cold.Commit}} · {{.Cold.GoVersion}} · {{.Cold.GOOS}}/{{.Cold.GOARCH}}</p></section>{{end}}
{{if .Scale}}<section class="panel"><h2>Startup by provider count</h2><p class="muted">Synthetic fixture: distinct providers without injected fields or HTTP routes. Warm runtime.New; numbers are measured and not deterministic.</p><table><thead><tr><th>Providers</th><th>Median (µs)</th><th>p95 (µs)</th><th>Bytes/op</th><th>Allocs/op</th></tr></thead><tbody>{{range .Scale}}<tr><td>{{.Providers}}</td><td>{{printf "%.2f" (micro .Startup.MedianNS)}}</td><td>{{printf "%.2f" (micro .Startup.P95NS)}}</td><td>{{.Startup.BytesPerOp}}</td><td>{{.Startup.AllocsPerOp}}</td></tr>{{end}}</tbody></table></section>{{end}}
<section class="panel"><h2>Module imports and exports</h2><p class="muted">Explicitly declared module metadata only. Imports point to known module types; exports refer to registered providers. Older generated modules have no import/export edges.</p>
<div class="metrics"><div class="metric"><strong>{{len .Modules.Modules}}</strong><span>Modules</span></div><div class="metric"><strong>{{len .Modules.Imports}}</strong><span>Module imports</span></div><div class="metric"><strong>{{len .Modules.Cycles}}</strong><span>Module cycles</span></div><div class="metric"><strong>{{len .Modules.Findings}}</strong><span>Module findings</span></div></div>
{{if .Modules.Imports}}<table><thead><tr><th>Module</th><th>Imports</th></tr></thead><tbody>{{range .Modules.Imports}}<tr><td><code>{{.From}}</code></td><td><code>{{.To}}</code></td></tr>{{end}}</tbody></table>{{end}}
{{if .Modules.Cycles}}<h3>Cyclic module components</h3><ul>{{range .Modules.Cycles}}<li class="warning"><code>{{join . " · "}}</code></li>{{end}}</ul>{{end}}
{{if .Modules.Findings}}<h3>Module diagnostics</h3><table><thead><tr><th>Kind</th><th>Module</th><th>Details</th></tr></thead><tbody>{{range .Modules.Findings}}<tr><td>{{.Kind}}</td><td><code>{{.Module}}</code></td><td>{{.Details}}</td></tr>{{end}}</tbody></table>{{end}}</section>
<section class="panel"><h2>Circular and unresolved dependencies</h2>
{{if .Graph.Cycles}}<ul>{{range .Graph.Cycles}}<li class="warning"><code>{{join . " · "}}</code></li>{{end}}</ul>{{else}}<p>No cycles found in the modeled provider graph.</p>{{end}}
{{if .Graph.Findings}}<table><thead><tr><th>Type</th><th>Provider</th><th>Field</th><th>Details</th></tr></thead><tbody>{{range .Graph.Findings}}<tr><td>{{.Kind}}</td><td><code>{{.Provider}}</code></td><td>{{.Field}}</td><td>{{.Details}}</td></tr>{{end}}</tbody></table>{{else}}<p>No unresolved, duplicate or ambiguous dependency findings.</p>{{end}}</section>
<section class="panel"><h2>Dependency edges</h2><table><thead><tr><th>From</th><th>Injection field</th><th>To</th></tr></thead><tbody>{{range .Graph.Edges}}<tr><td><code>{{.From}}</code></td><td>{{.Field}}</td><td><code>{{.To}}</code></td></tr>{{end}}</tbody></table></section>
<footer>Note: cycle components list participating providers, not an ordered exact cycle path. Time is nondeterministic and varies with environment and scheduler; only graph data has deterministic ordering.</footer>
</main></body></html>`))

func (r Report) HTML() ([]byte, error) {
	data := struct {
		Graph        Graph
		Modules      ModuleGraph
		Startup      Startup
		MedianUS     float64
		P95US        float64
		GraphSVG     template.HTML
		StartupSVG   template.HTML
		Cold         *ProcessStartup
		Scale        []ScaleStartup
		ColdMedianMS float64
		ColdP95MS    float64
	}{r.Graph, r.Modules, r.Startup, r.Startup.MedianNS / 1000, r.Startup.P95NS / 1000, graphSVG(r.Graph), startupSVG(r.Startup), r.Cold, r.Scale, 0, 0}
	if r.Cold != nil {
		data.ColdMedianMS = r.Cold.MedianNS / 1e6
		data.ColdP95MS = r.Cold.P95NS / 1e6
	}
	var out bytes.Buffer
	err := htmlTemplate.Execute(&out, data)
	return out.Bytes(), err
}
