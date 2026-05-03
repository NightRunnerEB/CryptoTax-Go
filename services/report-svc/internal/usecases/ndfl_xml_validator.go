package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type NDFLXMLValidator interface {
	Validate(ctx context.Context, payload []byte) error
}

type xmllintNDFLXMLValidator struct {
	schemaPath string
}

func NewNDFLXMLValidator(schemaPath string) (NDFLXMLValidator, error) {
	resolvedSchema, err := resolveNDFL3SchemaPath(schemaPath)
	if err != nil {
		return nil, err
	}
	return &xmllintNDFLXMLValidator{
		schemaPath: resolvedSchema,
	}, nil
}

func (v *xmllintNDFLXMLValidator) Validate(ctx context.Context, payload []byte) error {
	if len(payload) == 0 {
		return apperr.Internal("empty ndfl xml payload for validation", nil, nil)
	}

	cmd := exec.CommandContext(ctx, "xmllint", "--noout", "--nonet", "--schema", v.schemaPath, "-")
	cmd.Stdin = bytes.NewReader(payload)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		meta := map[string]string{
			"schema_path": v.schemaPath,
		}
		if msg := strings.TrimSpace(output.String()); msg != "" {
			meta["details"] = trimError(msg, 512)
		}
		if errors.Is(err, exec.ErrNotFound) {
			return apperr.Internal("xmllint is required for ndfl xsd validation", err, meta)
		}
		return apperr.RenderingFailed("ndfl xml does not satisfy xsd schema", err, meta)
	}

	return nil
}

func resolveNDFL3SchemaPath(preferred string) (string, error) {
	candidates := []string{
		strings.TrimSpace(preferred),
		strings.TrimSpace(os.Getenv("NDFL3_XSD_PATH")),
		"NDFL3.xsd.xml",
		filepath.Join("services", "report-svc", "NDFL3.xsd.xml"),
	}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "NDFL3.xsd.xml"))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		absPath, absErr := filepath.Abs(candidate)
		if absErr != nil {
			return "", apperr.Internal("resolve ndfl xsd path failed", absErr, map[string]string{
				"path": candidate,
			})
		}
		return absPath, nil
	}

	return "", apperr.Internal("ndfl xsd schema file not found", nil, map[string]string{
		"file": "NDFL3.xsd.xml",
	})
}

func trimError(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return fmt.Sprintf("%s...", value[:maxLen])
}
