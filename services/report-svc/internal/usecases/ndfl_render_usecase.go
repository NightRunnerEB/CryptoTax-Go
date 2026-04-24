package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type NDFLRenderUC struct {
	storage        domain.ObjectStorage
	validator      NDFLXMLValidator
	programVersion string
	formatVersion  string
}

func NewNDFLRenderUC(storage domain.ObjectStorage, programVersion string) (*NDFLRenderUC, error) {
	validator, err := NewNDFLXMLValidator("")
	if err != nil {
		return nil, err
	}
	return NewNDFLRenderUCWithValidator(storage, programVersion, validator), nil
}

func NewNDFLRenderUCWithValidator(storage domain.ObjectStorage, programVersion string, validator NDFLXMLValidator) *NDFLRenderUC {
	version := strings.TrimSpace(programVersion)
	if version == "" {
		version = "report-svc"
	}
	return &NDFLRenderUC{
		storage:        storage,
		validator:      validator,
		programVersion: version,
		formatVersion:  "5.20",
	}
}

func (u *NDFLRenderUC) Render(ctx context.Context, req domain.NDFLRenderRequest) (string, error) {
	if strings.TrimSpace(req.ReportID) == "" {
		return "", apperr.InvalidArgument("invalid report_id", nil, apperr.FieldViolation{
			Field:       "report_id",
			Description: "required",
		})
	}
	if strings.TrimSpace(req.UserID) == "" {
		return "", apperr.InvalidArgument("invalid user_id", nil, apperr.FieldViolation{
			Field:       "user_id",
			Description: "required",
		})
	}

	payload, err := buildNDFL3XML(req, u.programVersion, u.formatVersion)
	if err != nil {
		return "", err
	}
	if u.validator == nil {
		return "", apperr.Internal("ndfl xml validator is not configured", nil, nil)
	}
	if err := u.validator.Validate(ctx, payload); err != nil {
		return "", err
	}

	objectKey := fmt.Sprintf("reports/%s/%s.xml", strings.TrimSpace(req.UserID), strings.TrimSpace(req.ReportID))
	if err := u.storage.UploadXML(ctx, objectKey, payload); err != nil {
		return "", err
	}

	return objectKey, nil
}

var _ domain.NDFLRenderer = (*NDFLRenderUC)(nil)
