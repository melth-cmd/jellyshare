package handlers

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseBulkShareIDs(t *testing.T) {
	id := uuid.New()
	duplicate := []byte(`{"ids":["` + id.String() + `","` + id.String() + `"]}`)

	req := httptest.NewRequest("POST", "/api/admin/shares/bulk/revoke", bytes.NewReader(duplicate))
	rec := httptest.NewRecorder()
	ids, ok := parseBulkShareIDs(rec, req)
	if !ok {
		t.Fatalf("valid request rejected: %s", rec.Body.String())
	}
	if len(ids) != 1 || ids[0] != id {
		t.Errorf("ids = %v, want the one id once", ids)
	}

	req = httptest.NewRequest("POST", "/", strings.NewReader(`{"ids":[]}`))
	rec = httptest.NewRecorder()
	if _, ok := parseBulkShareIDs(rec, req); ok {
		t.Error("empty ids accepted")
	}
	if rec.Code != 400 {
		t.Errorf("empty ids status = %d, want 400", rec.Code)
	}

	var many strings.Builder
	many.WriteString(`{"ids":[`)
	for i := 0; i < maxBulkShareIDs+1; i++ {
		if i > 0 {
			many.WriteString(",")
		}
		many.WriteString(`"` + uuid.NewString() + `"`)
	}
	many.WriteString(`]}`)

	req = httptest.NewRequest("POST", "/", strings.NewReader(many.String()))
	rec = httptest.NewRecorder()
	if _, ok := parseBulkShareIDs(rec, req); ok {
		t.Error("too many ids accepted")
	}
	if rec.Code != 400 {
		t.Errorf("too many ids status = %d, want 400", rec.Code)
	}
}
