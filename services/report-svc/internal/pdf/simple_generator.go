package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

const (
	maxLinesPerPage = 45
)

type SimpleGenerator struct{}

func NewSimpleGenerator() *SimpleGenerator {
	return &SimpleGenerator{}
}

func (g *SimpleGenerator) Generate(dataset domain.ReportDataset, opts domain.PDFOptions) ([]byte, error) {
	if strings.TrimSpace(dataset.ReportID) == "" {
		return nil, apperr.RenderingFailed("dataset report_id is required", nil, nil)
	}
	if strings.TrimSpace(dataset.UserID) == "" {
		return nil, apperr.RenderingFailed("dataset user_id is required", nil, nil)
	}

	previewLimit := opts.MaxPreviewRows
	if previewLimit <= 0 {
		previewLimit = 20
	}

	lines := make([]string, 0, 64)
	lines = append(lines, "Tax Report")
	lines = append(lines, fmt.Sprintf("Report ID: %s", dataset.ReportID))
	lines = append(lines, fmt.Sprintf("User: %s", dataset.UserID))
	lines = append(lines, fmt.Sprintf("Year: %d", dataset.TaxYear))
	lines = append(lines, fmt.Sprintf("Jurisdiction: %s", dataset.Jurisdiction))
	if strings.TrimSpace(opts.TemplateVersion) != "" {
		lines = append(lines, fmt.Sprintf("Template Version: %s", opts.TemplateVersion))
	}
	lines = append(lines, "")
	lines = append(lines, "Taxpayer")
	lines = append(lines, fmt.Sprintf("INN: %s", orDash(dataset.TaxpayerProfile.INN)))
	lines = append(lines, fmt.Sprintf("Name: %s %s %s",
		orDash(dataset.TaxpayerProfile.LastName),
		orDash(dataset.TaxpayerProfile.FirstName),
		orDash(dataset.TaxpayerProfile.MiddleName),
	))
	lines = append(lines, fmt.Sprintf("Birth Date: %s", orDash(dataset.TaxpayerProfile.BirthDate)))
	lines = append(lines, "")
	lines = append(lines, "Summary")
	lines = append(lines, renderSummaryLines(dataset.Summary)...)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Total events: %d", len(dataset.Events)))
	lines = append(lines, fmt.Sprintf("Events preview (max %d):", previewLimit))

	for i, event := range dataset.Events {
		if i >= previewLimit {
			break
		}
		lines = append(lines, fmt.Sprintf(
			"%s | %s | %s | %s %s | fiat=%s",
			event.TimeUTC.UTC().Format(time.RFC3339),
			compact(event.EventType),
			compact(event.TxID),
			compact(event.AssetSymbol),
			compact(event.CryptoAmount),
			compact(ptrString(event.FiatAmount)),
		))
	}

	if len(lines) > maxLinesPerPage {
		lines = lines[:maxLinesPerPage]
	}
	content := buildContentStream(lines)
	return buildSinglePagePDF(content), nil
}

func buildContentStream(lines []string) string {
	var b strings.Builder
	b.WriteString("BT\n")
	b.WriteString("/F1 10 Tf\n")
	b.WriteString("13 TL\n")
	b.WriteString("40 800 Td\n")

	for i, line := range lines {
		if i > 0 {
			b.WriteString("T*\n")
		}
		b.WriteString("(")
		b.WriteString(escapePDFText(toASCII(line)))
		b.WriteString(") Tj\n")
	}
	b.WriteString("ET\n")
	return b.String()
}

func buildSinglePagePDF(content string) []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")

	offsets := make([]int, 0, len(objects))
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		out.WriteString(fmt.Sprintf("%d 0 obj\n", i+1))
		out.WriteString(obj)
		out.WriteString("\nendobj\n")
	}

	xrefOffset := out.Len()
	out.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	out.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		out.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	out.WriteString("trailer\n")
	out.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	out.WriteString("startxref\n")
	out.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	out.WriteString("%%EOF")
	return out.Bytes()
}

func renderSummaryLines(summary map[string]any) []string {
	if len(summary) == 0 {
		return []string{"summary: n/a"}
	}
	keys := make([]string, 0, len(summary))
	for k := range summary {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %v", key, summary[key]))
	}
	return lines
}

func escapePDFText(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, "(", `\(`)
	v = strings.ReplaceAll(v, ")", `\)`)
	return v
}

func toASCII(v string) string {
	if v == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if r >= 32 && r <= 126 {
			b.WriteRune(r)
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		b.WriteByte('?')
	}
	return b.String()
}

func compact(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func orDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

var _ domain.PDFGenerator = (*SimpleGenerator)(nil)
