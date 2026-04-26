package accountpool

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeHandlerService struct {
	putRouteFn func(ctx *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error)
}

func (f *fakeHandlerService) ListPools(_ *gin.Context) ([]*Pool, error) { return nil, nil }
func (f *fakeHandlerService) GetPool(_ *gin.Context, _ uint64) (*Pool, error) {
	return nil, nil
}
func (f *fakeHandlerService) CreatePool(_ *gin.Context, _ CreatePoolInput) (*Pool, error) {
	return nil, nil
}
func (f *fakeHandlerService) UpdatePool(_ *gin.Context, _ uint64, _ UpdatePoolInput) (*Pool, error) {
	return nil, nil
}
func (f *fakeHandlerService) DeletePool(_ *gin.Context, _ uint64) error { return nil }
func (f *fakeHandlerService) ListMembers(_ *gin.Context, _ uint64) ([]*Member, error) {
	return nil, nil
}
func (f *fakeHandlerService) UpsertMember(_ *gin.Context, _ uint64, _ uint64, _ UpsertMemberInput) (*Member, error) {
	return nil, nil
}
func (f *fakeHandlerService) DeleteMember(_ *gin.Context, _, _ uint64) error { return nil }
func (f *fakeHandlerService) ListRoutes(_ *gin.Context) ([]*ModelRoute, error) { return nil, nil }
func (f *fakeHandlerService) PutRoute(ctx *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error) {
	return f.putRouteFn(ctx, modelID, in)
}
func (f *fakeHandlerService) DeleteRoute(_ *gin.Context, _ uint64) error { return nil }

func TestPutRouteCreatesOrUpdatesModelRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeHandlerService{
		putRouteFn: func(_ *gin.Context, modelID uint64, in PutRouteInput) (*ModelRoute, error) {
			return &ModelRoute{
				ModelID:        modelID,
				PoolID:         in.PoolID,
				FallbackPoolID: in.FallbackPoolID,
				Enabled:        true,
			}, nil
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.PUT("/routes/:modelId", h.PutRoute)

	req := httptest.NewRequest(http.MethodPut, "/routes/9", strings.NewReader(`{"pool_id":1,"fallback_pool_id":2}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"pool_id":1`) {
		t.Fatalf("expected pool_id in response body, got %s", w.Body.String())
	}
}
