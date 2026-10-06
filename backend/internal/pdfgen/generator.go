package pdfgen

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/markdown"
)

type Metadata struct {
	SOPTitle         string
	SOPID            string
	SOPVersionID     string
	SOPVersionNumber int
	Stage            string
	ContentHash      string
	GeneratorVersion string
	GeneratedAt      time.Time
	Locale           string
}

type Input struct {
	Markdown string
	Meta     Metadata
}

type Renderer interface {
	Render(input Input) ([]byte, error)
}

type GotenbergRenderer struct {
	baseURL string
	client  *http.Client
}

//go:embed assets/favicon.svg
var faviconSVG []byte

func NewGotenbergRenderer(baseURL string, timeout time.Duration) *GotenbergRenderer {
	return &GotenbergRenderer{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (g *GotenbergRenderer) Render(input Input) ([]byte, error) {
	var mdHTML bytes.Buffer
	md := markdown.AuthoringMarkdown()
	if err := md.Convert([]byte(input.Markdown), &mdHTML); err != nil {
		return nil, fmt.Errorf("failed to convert markdown to html: %w", err)
	}

	p := bluemonday.UGCPolicy()
	p.AllowDataURIImages()
	safeHTML := p.SanitizeBytes(mdHTML.Bytes())
	view := documentView(input, template.HTML(string(safeHTML)))

	tmpl, err := template.New("pdf").Parse(pdfTemplate)
	if err != nil {
		return nil, err
	}
	var htmlDoc bytes.Buffer
	if err := tmpl.Execute(&htmlDoc, view); err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(htmlDoc.Bytes()); err != nil {
		return nil, err
	}
	headerTmpl, err := buildChromeTemplate(chromeHeaderTemplate, view)
	if err != nil {
		return nil, err
	}
	headerPart, err := writer.CreateFormFile("files", "header.html")
	if err != nil {
		return nil, err
	}
	if _, err := headerPart.Write([]byte(headerTmpl)); err != nil {
		return nil, err
	}
	footerTmpl, err := buildChromeTemplate(chromeFooterTemplate, view)
	if err != nil {
		return nil, err
	}
	footerPart, err := writer.CreateFormFile("files", "footer.html")
	if err != nil {
		return nil, err
	}
	if _, err := footerPart.Write([]byte(footerTmpl)); err != nil {
		return nil, err
	}
	if err := writer.WriteField("displayHeaderFooter", "true"); err != nil {
		return nil, err
	}
	if err := writer.WriteField("marginTop", "1.1"); err != nil {
		return nil, err
	}
	if err := writer.WriteField("marginBottom", "0.8"); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := g.baseURL + "/forms/chromium/convert/html"
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gotenberg request failed: %w", err)
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(pdfBytes))
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		return nil, fmt.Errorf("gotenberg error (%d): %s", resp.StatusCode, snippet)
	}
	return pdfBytes, nil
}

func ValidateBaseURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("url is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("url must include scheme and host")
	}
	return nil
}

func buildChromeTemplate(tmplText string, data any) (string, error) {
	tmpl, err := template.New("chromium-part").Parse(tmplText)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func stageLabel(locale, stage string) string {
	key := "pdf.stage." + stage
	label := i18n.T(locale, key, nil)
	if label == key {
		return stage
	}
	return label
}

func documentView(input Input, content template.HTML) pdfView {
	locale := i18n.Fallback(input.Meta.Locale)
	stage := stageLabel(locale, input.Meta.Stage)
	version := fmt.Sprintf("%d", input.Meta.SOPVersionNumber)
	return pdfView{
		Lang:                 locale,
		Title:                input.Meta.SOPTitle,
		VersionNumber:        input.Meta.SOPVersionNumber,
		Stage:                stage,
		SOPID:                input.Meta.SOPID,
		VersionID:            input.Meta.SOPVersionID,
		ContentHash:          input.Meta.ContentHash,
		Generator:            input.Meta.GeneratorVersion,
		GeneratedAt:          input.Meta.GeneratedAt.UTC().Format(time.RFC3339),
		GeneratedAtDisplay:   input.Meta.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"),
		FaviconDataURL:       "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(faviconSVG),
		FaviconInlineSVG:     template.HTML(string(faviconSVG)),
		ConfidentialityLabel: i18n.T(locale, "pdf.confidential", nil),
		TechnicalInformation: i18n.T(locale, "pdf.technical_information", nil),
		LifecycleStatus:      i18n.T(locale, "pdf.lifecycle_status", nil),
		VersionLine:          i18n.T(locale, "pdf.version_line", map[string]string{"version": version, "stage": stage}),
		SOPIDLabel:           i18n.T(locale, "pdf.sop_id", nil),
		SOPVersionLabel:      i18n.T(locale, "pdf.sop_version", nil),
		VersionIDLabel:       i18n.T(locale, "pdf.version_id", nil),
		ContentHashLabel:     i18n.T(locale, "pdf.content_hash", nil),
		GeneratedAtLabel:     i18n.T(locale, "pdf.generated_at", nil),
		PageLabel:            i18n.T(locale, "pdf.page", nil),
		ContentHTML:          content,
	}
}

// FooterHTML renders the PDF footer chrome, including the confidentiality label.
func FooterHTML(input Input) (string, error) {
	view := documentView(input, "")
	return buildChromeTemplate(chromeFooterTemplate, view)
}

// HTMLDocument renders the SOP HTML shell, including translated chrome.
func HTMLDocument(input Input) (string, error) {
	var mdHTML bytes.Buffer
	md := markdown.AuthoringMarkdown()
	if err := md.Convert([]byte(input.Markdown), &mdHTML); err != nil {
		return "", fmt.Errorf("failed to convert markdown to html: %w", err)
	}
	p := bluemonday.UGCPolicy()
	p.AllowDataURIImages()
	view := documentView(input, template.HTML(string(p.SanitizeBytes(mdHTML.Bytes()))))
	tmpl, err := template.New("pdf").Parse(pdfTemplate)
	if err != nil {
		return "", err
	}
	var htmlDoc bytes.Buffer
	if err := tmpl.Execute(&htmlDoc, view); err != nil {
		return "", err
	}
	return htmlDoc.String(), nil
}

type pdfView struct {
	Lang                 string
	Title                string
	VersionNumber        int
	Stage                string
	SOPID                string
	VersionID            string
	ContentHash          string
	Generator            string
	GeneratedAt          string
	GeneratedAtDisplay   string
	FaviconDataURL       string
	FaviconInlineSVG     template.HTML
	ConfidentialityLabel string
	TechnicalInformation string
	LifecycleStatus      string
	VersionLine          string
	SOPIDLabel           string
	SOPVersionLabel      string
	VersionIDLabel       string
	ContentHashLabel     string
	GeneratedAtLabel     string
	PageLabel            string
	ContentHTML          template.HTML
}

const pdfTemplate = `<!doctype html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: Arial, sans-serif; font-size: 12px; color: #1f2937; margin: 30px; line-height: 1.5; }
    h1 { margin: 0 0 8px 0; font-size: 24px; }
    .sub { margin-bottom: 18px; color: #4b5563; }
    .title-row { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
    .title-row .favicon { width: 20px; height: 20px; display: inline-flex; align-items: center; justify-content: center; }
    .title-row .favicon svg { width: 20px; height: 20px; display: block; }
    .title-row h1 { margin: 0; }
    .meta-heading { margin: 0 0 8px 0; font-size: 14px; color: #111827; }
    .meta-table { font-size: 11px; margin: 0 0 18px 0; }
    .meta-table th { width: 220px; text-align: left; background: #f9fafb; color: #374151; }
    .meta-table td { word-break: break-word; }
    table { width: 100%; border-collapse: collapse; margin: 12px 0; }
    th, td { border: 1px solid #d1d5db; padding: 6px; vertical-align: top; }
    img { max-width: 100%; height: auto; }
    code, pre { background: #f3f4f6; }
    pre { padding: 8px; overflow-x: auto; }
  </style>
</head>
<body>
  <div class="title-row">
    <h1>{{.Title}}</h1>
  </div>
  <div class="sub">{{.VersionLine}}</div>
  <h2 class="meta-heading">{{.TechnicalInformation}}</h2>
  <table class="meta-table">
    <tbody>
      <tr><th>{{.SOPIDLabel}}</th><td>{{.SOPID}}</td></tr>
      <tr><th>{{.SOPVersionLabel}}</th><td>{{.VersionNumber}}</td></tr>
      <tr><th>{{.LifecycleStatus}}</th><td>{{.Stage}}</td></tr>
      <tr><th>{{.VersionIDLabel}}</th><td>{{.VersionID}}</td></tr>
      <tr><th>{{.ContentHashLabel}}</th><td>{{.ContentHash}}</td></tr>
      <tr><th>{{.GeneratedAtLabel}}</th><td>{{.GeneratedAt}}</td></tr>
    </tbody>
  </table>
  <article>{{.ContentHTML}}</article>
</body>
</html>`

const chromeHeaderTemplate = `<div style="width:100%; font-size:9px; color:#4b5563; padding:0 24px 0 24px; box-sizing:border-box; border-bottom:1px solid #e5e7eb;">
  <div style="display:flex; align-items:center; justify-content:space-between; width:100%; min-height:28px;">
    <div style="display:flex; align-items:center; gap:6px;">
      <span style="width:24px; height:24px; display:inline-flex; align-items:center; justify-content:center;">{{.FaviconInlineSVG}}</span>
      <div>
        <div style="font-size:10px; color:#111827; font-weight:600;">{{.Title}}</div>
        <div>{{.SOPID}} v{{.VersionNumber}} ({{.Stage}})</div>
      </div>
    </div>
    <div>{{.GeneratedAtDisplay}}</div>
  </div>
</div>`

const chromeFooterTemplate = `<div style="width:100%; font-size:9px; color:#6b7280; padding:0 24px 0 24px; box-sizing:border-box; border-top:1px solid #e5e7eb;">
  <div style="display:flex; align-items:center; justify-content:space-between; width:100%; min-height:24px;">
    <div>{{.ConfidentialityLabel}}</div>
    <div>{{.PageLabel}} <span class="pageNumber"></span> / <span class="totalPages"></span></div>
  </div>
</div>`
