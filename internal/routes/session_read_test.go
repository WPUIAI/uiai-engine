package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestSessionReadRouteSupportsCanonicalGETAndCompatiblePOST(t *testing.T) {
	router := chi.NewRouter()
	manager := &vision.SessionManager{}
	router.Route("/api/session", func(r chi.Router) {
		MountSessionRoutes(r, nil, manager)
	})

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/session/missing/read?max_chars=12000&include_links=true", nil)
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf("GET /read status = %d, want route-owned 404; body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	if getRecorder.Code == http.StatusMethodNotAllowed {
		t.Fatal("documented GET /read returned 405")
	}
	if value := getRecorder.Header().Get("Deprecation"); value != "" {
		t.Fatalf("canonical GET marked deprecated: %q", value)
	}

	postRecorder := httptest.NewRecorder()
	postRequest := httptest.NewRequest(http.MethodPost, "/api/session/missing/read", nil)
	router.ServeHTTP(postRecorder, postRequest)
	if postRecorder.Code != http.StatusNotFound {
		t.Fatalf("POST /read status = %d, want compatibility-route 404; body=%s", postRecorder.Code, postRecorder.Body.String())
	}
	if postRecorder.Header().Get("Deprecation") != "true" {
		t.Fatal("compatibility POST /read lacks deprecation header")
	}
	if postRecorder.Header().Get("Warning") == "" {
		t.Fatal("compatibility POST /read lacks migration warning")
	}
}

func TestReadOptionsFromQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/read?selector=%23main&max_chars=12000&include_links=true&format=markdown&mode=article&include_images=true", nil)
	options, err := readOptionsFromQuery(req)
	if err != nil {
		t.Fatal(err)
	}
	if options.Selector != "#main" || options.MaxChars != 12000 || !options.IncludeLinks || options.Format != "markdown" || options.Mode != "article" || !options.IncludeImages {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestReadOptionsFromQueryRejectsMalformedBounds(t *testing.T) {
	for _, query := range []string{
		"max_chars=not-a-number",
		"max_chars=-1",
		"include_links=maybe",
		"include_images=maybe",
	} {
		t.Run(query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/read?"+query, nil)
			if _, err := readOptionsFromQuery(req); err == nil {
				t.Fatal("malformed query accepted")
			}
		})
	}
}

func TestEmbeddedOpenAPICanonicalizesSessionReadGET(t *testing.T) {
	recorder := httptest.NewRecorder()
	HandleOpenAPI(recorder, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("OpenAPI status = %d", recorder.Code)
	}
	var document struct {
		Paths map[string]map[string]struct {
			Deprecated bool `json:"deprecated"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	readPath := document.Paths["/api/session/{id}/read"]
	if _, ok := readPath["get"]; !ok {
		t.Fatal("OpenAPI omits canonical GET /api/session/{id}/read")
	}
	post, ok := readPath["post"]
	if !ok || !post.Deprecated {
		t.Fatal("OpenAPI omits deprecated POST /read compatibility declaration")
	}
}
