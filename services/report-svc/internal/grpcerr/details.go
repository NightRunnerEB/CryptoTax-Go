package grpcerr

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoreflect"

	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

func toDetails(ae *apperr.Error, domain string) []protoadapt.MessageV1 {
	out := make([]protoadapt.MessageV1, 0, 1+len(ae.Details))

	out = append(out, v1(&errdetails.ErrorInfo{
		Reason:   string(ae.Code),
		Domain:   domain,
		Metadata: ae.Meta,
	}))

	out = append(out, mapDetails(ae.Details)...)
	return out
}

func mapDetails(details []apperr.Detail) []protoadapt.MessageV1 {
	if len(details) == 0 {
		return nil
	}

	out := make([]protoadapt.MessageV1, 0, len(details))
	for _, d := range details {
		switch v := d.(type) {
		case apperr.Validation:
			br := &errdetails.BadRequest{}
			for _, fv := range v.Violations {
				br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fv.Field,
					Description: fv.Description,
				})
			}
			out = append(out, v1(br))
		case apperr.Resource:
			out = append(out, v1(&errdetails.ResourceInfo{
				ResourceType: v.Type,
				ResourceName: v.Name,
			}))
		}
	}

	return out
}

func v1(m protoreflect.ProtoMessage) protoadapt.MessageV1 {
	return protoadapt.MessageV1Of(m)
}
