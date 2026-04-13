package consumer

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

func TestDecodeImportCompletedEvent(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	importID := uuid.New()
	eventID := uuid.New()

	t.Run("envelope format", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"ImportCompleted":{"event_id":"` + eventID.String() + `","user_id":"` + userID.String() + `","import_id":"` + importID.String() + `"}}`)
		event, err := decodeImportCompletedEvent(body)
		if err != nil {
			t.Fatalf("decodeImportCompletedEvent returned error: %v", err)
		}
		if event.UserID != userID || event.ImportID != importID || event.EventId != eventID {
			t.Fatalf("unexpected event: %+v", event)
		}
	})

	t.Run("flat format", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"event_id":"` + eventID.String() + `","user_id":"` + userID.String() + `","import_id":"` + importID.String() + `"}`)
		event, err := decodeImportCompletedEvent(body)
		if err != nil {
			t.Fatalf("decodeImportCompletedEvent returned error: %v", err)
		}
		if event.UserID != userID || event.ImportID != importID || event.EventId != eventID {
			t.Fatalf("unexpected event: %+v", event)
		}
	})

	t.Run("unsupported import event type", func(t *testing.T) {
		t.Parallel()
		body := []byte(`{"ImportCreated":{"event_id":"` + eventID.String() + `","user_id":"` + userID.String() + `","import_id":"` + importID.String() + `"}}`)
		_, err := decodeImportCompletedEvent(body)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestShouldRequeue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "invalid argument does not requeue",
			err:  apperr.InvalidArgument("bad payload", nil, apperr.FieldViolation{Field: "user_id", Description: "required"}),
			want: false,
		},
		{
			name: "ledger unavailable requeues",
			err:  apperr.LedgerUnavailable("ledger unavailable", nil, nil),
			want: true,
		},
		{
			name: "ledger bad response 404 does not requeue",
			err:  apperr.LedgerBadResponse("ledger bad response", nil, map[string]string{"status_code": "404"}),
			want: false,
		},
		{
			name: "ledger bad response 503 requeues",
			err:  apperr.LedgerBadResponse("ledger bad response", nil, map[string]string{"status_code": "503"}),
			want: true,
		},
		{
			name: "import already completed does not requeue",
			err:  apperr.ImportAlreadyDone("already done", nil, nil),
			want: false,
		},
		{
			name: "non-domain error requeues",
			err:  errors.New("unexpected"),
			want: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldRequeue(tc.err); got != tc.want {
				t.Fatalf("unexpected shouldRequeue result: got=%v want=%v", got, tc.want)
			}
		})
	}
}
