package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Liveness(t *testing.T) {
	handler := NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	rr := httptest.NewRecorder()

	handler.Liveness(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Liveness должен возвращать статус 200 OK")
	assert.Equal(t, "OK", rr.Body.String(), "Тело ответа должно быть 'OK'")
}
