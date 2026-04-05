package report

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

// PDFWriter implements app.ReportWriter using the maroto v2 library.
type PDFWriter struct {
	reportsDir string
}

// NewPDFWriter creates a writer that stores PDFs under reportsDir.
func NewPDFWriter(reportsDir string) *PDFWriter {
	return &PDFWriter{reportsDir: reportsDir}
}

// Write generates a PDF from the AnalysisResult and returns the file path.
func (w *PDFWriter) Write(_ context.Context, result *domain.AnalysisResult) (string, error) {
	m, err := w.buildDocument(result)
	if err != nil {
		return "", fmt.Errorf("build PDF: %w", err)
	}

	doc, err := m.Generate()
	if err != nil {
		return "", fmt.Errorf("generate PDF: %w", err)
	}

	outPath := w.outputPath(result)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return "", fmt.Errorf("create report dir: %w", err)
	}

	if err := doc.Save(outPath); err != nil {
		return "", fmt.Errorf("save PDF: %w", err)
	}

	return outPath, nil
}

func (w *PDFWriter) buildDocument(result *domain.AnalysisResult) (core.Maroto, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(PageMarginLeft).
		WithRightMargin(PageMarginRight).
		WithTopMargin(PageMarginTop).
		WithBottomMargin(PageMarginBottom).
		Build()

	m := maroto.New(cfg)

	addHeader(m, result)
	addExecutiveSummary(m, result)
	addPerProfileFindings(m, result)
	addConsolidatedAnalysis(m, result)
	addRecommendations(m, result)
	addFooter(m, result)

	return m, nil
}

// --- Section builders ---

func addHeader(m core.Maroto, r *domain.AnalysisResult) {
	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New("pprof-analyzer — Performance Report", props.Text{
					Size:  FontSizeTitle,
					Style: fontstyle.Bold,
					Align: align.Center,
					Color: hexToColor(ColorDark),
				}),
			),
		),
		row.New(6).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Application: %s | Environment: %s | Collected: %s",
					r.EndpointName,
					strings.ToUpper(string(r.Environment)),
					r.CollectedAt.Format("2006-01-02 15:04:05 UTC"),
				), props.Text{
					Size:  FontSizeBody,
					Align: align.Center,
					Color: hexToColor(ColorGray),
				}),
			),
		),
		row.New(4).Add(col.New(12)), // spacer
	)
}

func addExecutiveSummary(m core.Maroto, r *domain.AnalysisResult) {
	severityColor := hexToColor(SeverityColor(r.OverallSeverity))
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("Executive Summary", props.Text{
					Size:  FontSizeSection,
					Style: fontstyle.Bold,
					Color: hexToColor(ColorDark),
				}),
			),
		),
		row.New(6).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Overall Severity: %s", SeverityLabel(r.OverallSeverity)), props.Text{
					Size:  FontSizeBody,
					Style: fontstyle.Bold,
					Color: severityColor,
				}),
			),
		),
		row.New(10).Add(
			col.New(12).Add(
				text.New(r.ExecutiveSummary, props.Text{
					Size: FontSizeBody,
				}),
			),
		),
		row.New(4).Add(col.New(12)), // spacer
	)
}

func addPerProfileFindings(m core.Maroto, r *domain.AnalysisResult) {
	if len(r.PerProfileFindings) == 0 {
		return
	}

	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("Analysis by Profile", props.Text{
					Size:  FontSizeSection,
					Style: fontstyle.Bold,
					Color: hexToColor(ColorDark),
				}),
			),
		),
	)

	for _, f := range r.PerProfileFindings {
		sColor := hexToColor(SeverityColor(f.Severity))
		m.AddRows(
			row.New(6).Add(
				col.New(8).Add(
					text.New(ProfileTypeLabel(f.ProfileType), props.Text{
						Size:  FontSizeBody,
						Style: fontstyle.Bold,
					}),
				),
				col.New(4).Add(
					text.New(SeverityLabel(f.Severity), props.Text{
						Size:  FontSizeBody,
						Style: fontstyle.Bold,
						Color: sColor,
						Align: align.Right,
					}),
				),
			),
			row.New(6).Add(
				col.New(12).Add(
					text.New(f.Summary, props.Text{
						Size:  FontSizeBody,
						Style: fontstyle.Italic,
					}),
				),
			),
			row.New(10).Add(
				col.New(12).Add(
					text.New(f.Details, props.Text{
						Size: FontSizeBody,
					}),
				),
			),
			row.New(3).Add(col.New(12)), // spacer
		)
	}
}

func addConsolidatedAnalysis(m core.Maroto, r *domain.AnalysisResult) {
	if r.ConsolidatedAnalysis == "" {
		return
	}

	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("Consolidated Analysis", props.Text{
					Size:  FontSizeSection,
					Style: fontstyle.Bold,
					Color: hexToColor(ColorDark),
				}),
			),
		),
		row.New(20).Add(
			col.New(12).Add(
				text.New(r.ConsolidatedAnalysis, props.Text{
					Size: FontSizeBody,
				}),
			),
		),
		row.New(4).Add(col.New(12)), // spacer
	)
}

func addRecommendations(m core.Maroto, r *domain.AnalysisResult) {
	if len(r.Recommendations) == 0 {
		return
	}

	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("Recommendations", props.Text{
					Size:  FontSizeSection,
					Style: fontstyle.Bold,
					Color: hexToColor(ColorDark),
				}),
			),
		),
	)

	for _, rec := range r.Recommendations {
		m.AddRows(
			row.New(6).Add(
				col.New(12).Add(
					text.New(fmt.Sprintf("%d. %s", rec.Priority, rec.Title), props.Text{
						Size:  FontSizeBody,
						Style: fontstyle.Bold,
					}),
				),
			),
			row.New(8).Add(
				col.New(12).Add(
					text.New(rec.Description, props.Text{
						Size: FontSizeBody,
					}),
				),
			),
		)

		if rec.CodeSuggestion != "" {
			m.AddRows(
				row.New(8).Add(
					col.New(12).Add(
						text.New(rec.CodeSuggestion, props.Text{
							Size:  FontSizeCode,
							Style: fontstyle.Normal,
							Color: hexToColor(ColorGray),
						}),
					),
				),
			)
		}

		m.AddRows(row.New(3).Add(col.New(12)))
	}
}

func addFooter(m core.Maroto, r *domain.AnalysisResult) {
	m.AddRows(
		row.New(4).Add(col.New(12)), // spacer
		row.New(6).Add(
			col.New(12).Add(
				text.New(
					fmt.Sprintf("Generated by pprof-analyzer %s | Model: %s | %s",
						r.ToolVersion, r.ModelUsed, time.Now().UTC().Format("2006-01-02 15:04:05 UTC")),
					props.Text{
						Size:  FontSizeSmall,
						Align: align.Center,
						Color: hexToColor(ColorGray),
					},
				),
			),
		),
	)
}

// outputPath returns the full path for the PDF file.
// Pattern: {reportsDir}/{app}/{env}/{YYYY-MM-DD}/{app}_{env}_{YYYYMMDD_HHMMSS}.pdf
func (w *PDFWriter) outputPath(r *domain.AnalysisResult) string {
	dateDir := r.CollectedAt.UTC().Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s_%s.pdf",
		sanitize(r.EndpointName),
		sanitize(string(r.Environment)),
		r.CollectedAt.UTC().Format("20060102_150405"),
	)
	return filepath.Join(w.reportsDir, sanitize(r.EndpointName), string(r.Environment), dateDir, filename)
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "-")
}

// hexToColor converts a hex color string (#RRGGBB) to a maroto props.Color.
func hexToColor(hex string) *props.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return &props.Color{Red: 0, Green: 0, Blue: 0}
	}
	var r, g, b int
	fmt.Sscanf(hex[:2], "%x", &r)
	fmt.Sscanf(hex[2:4], "%x", &g)
	fmt.Sscanf(hex[4:6], "%x", &b)
	return &props.Color{Red: r, Green: g, Blue: b}
}
