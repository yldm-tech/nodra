package devtools

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/yldm-tech/nodra"
)

func request(i *Inspector, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:12345"+path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+i.token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	i.Handler().ServeHTTP(w, r)
	return w
}

func TestHTTPInspectionExportRestoreAndClear(t *testing.T) {
	i := New(Options{AllowRestore: true})
	defer func() { _ = i.Close() }()
	store := nodra.New(counterState{})
	mustAttach(t, i, "counter", store)
	response := request(i, "GET", "/api/snapshot", "")
	if response.Code != http.StatusOK {
		t.Fatal(response.Code)
	}
	var snapshot Snapshot
	mustUpdate(t, json.Unmarshal(response.Body.Bytes(), &snapshot))
	if len(snapshot.Stores) != 1 || snapshot.Format != 1 {
		t.Fatalf("export = %+v", snapshot)
	}
	response = request(i, "POST", "/api/restore", `{"storeId":"counter","state":{"count":7}}`)
	if response.Code != http.StatusNoContent || store.Get().Count != 7 {
		t.Fatalf("apply = %d %s", response.Code, response.Body.String())
	}
	response = request(i, "POST", "/api/restore", `{"storeId":"counter","sequence":1}`)
	if response.Code != http.StatusNoContent || store.Get().Count != 0 {
		t.Fatalf("restore = %d %s", response.Code, response.Body.String())
	}
	if response := request(i, "POST", "/api/clear", "{}"); response.Code != http.StatusNoContent || len(i.Snapshot().Events) != 0 {
		t.Fatal("timeline did not clear")
	}
}

func TestHTTPRejectsUnauthorizedAndCrossOriginRequests(t *testing.T) {
	i := New(Options{})
	defer func() { _ = i.Close() }()
	for _, test := range []struct {
		name, host, origin, auth string
		status                   int
	}{
		{"missing-token", "127.0.0.1:12345", "", "", http.StatusUnauthorized},
		{"wrong-token", "127.0.0.1:12345", "", "Bearer wrong", http.StatusUnauthorized},
		{"missing-bearer", "127.0.0.1:12345", "", i.token, http.StatusUnauthorized},
		{"dns-rebinding", "attacker.invalid:12345", "", "Bearer " + i.token, http.StatusForbidden},
		{"cross-origin", "127.0.0.1:12345", "https://attacker.invalid", "Bearer " + i.token, http.StatusForbidden},
		{"same-origin", "127.0.0.1:12345", "http://127.0.0.1:12345", "Bearer " + i.token, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://127.0.0.1:12345/api/snapshot", nil)
			r.Host = test.host
			r.Header.Set("Authorization", test.auth)
			r.Header.Set("Origin", test.origin)
			w := httptest.NewRecorder()
			i.Handler().ServeHTTP(w, r)
			if w.Code != test.status {
				t.Fatalf("status = %d, want %d", w.Code, test.status)
			}
		})
	}
}

func TestHTTPRejectsInvalidRestoresAndMethods(t *testing.T) {
	i := New(Options{AllowRestore: true})
	defer func() { _ = i.Close() }()
	store := nodra.New(counterState{})
	mustAttach(t, i, "counter", store)
	for _, test := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/api/restore", "", http.StatusMethodNotAllowed},
		{"POST", "/api/snapshot", "{}", http.StatusMethodNotAllowed},
		{"GET", "/api/unknown", "", http.StatusNotFound},
		{"POST", "/api/restore", `{}`, http.StatusBadRequest},
		{"POST", "/api/restore", `{"storeId":"counter","state":{},"sequence":1}`, http.StatusBadRequest},
		{"POST", "/api/restore", `{"storeId":"counter","state":{"unknown":4}}`, http.StatusBadRequest},
		{"POST", "/api/restore", `{"storeId":"counter","state":{"count":"wrong"}}`, http.StatusBadRequest},
		{"POST", "/api/restore", `{"storeId":"counter","sequence":999}`, http.StatusNotFound},
		{"POST", "/api/restore", `{"storeId":"missing","state":{}}`, http.StatusNotFound},
		{"POST", "/api/restore", strings.Repeat("x", (1<<20)+1), http.StatusRequestEntityTooLarge},
	} {
		if w := request(i, test.method, test.path, test.body); w.Code != test.status {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, w.Code, test.status)
		}
	}
	if store.Get().Count != 0 || store.Version() != 0 {
		t.Fatal("invalid HTTP restore modified state")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1/api/restore", strings.NewReader(`{}`))
	r.Header.Set("Authorization", "Bearer "+i.token)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	i.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatal(w.Code)
	}
	readonly := New(Options{})
	defer func() { _ = readonly.Close() }()
	mustAttach(t, readonly, "counter", store)
	if w := request(readonly, "POST", "/api/restore", `{"storeId":"counter","state":{}}`); w.Code != http.StatusForbidden {
		t.Fatal("HTTP restore is not opt-in")
	}
}

func TestHTTPTokenRotationInvalidatesOldCapability(t *testing.T) {
	i := New(Options{})
	defer func() { _ = i.Close() }()
	old := i.token
	newToken, err := i.RotateToken()
	if err != nil || newToken == "" || newToken == old {
		t.Fatalf("rotation old=%q new=%q err=%v", old, newToken, err)
	}
	requestWithToken := func(token string) int {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/api/snapshot", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		i.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if got := requestWithToken(old); got != http.StatusUnauthorized {
		t.Fatalf("old token status=%d", got)
	}
	if got := requestWithToken(newToken); got != http.StatusOK {
		t.Fatalf("new token status=%d", got)
	}
}

func TestHTTPRateLimitAppliesAfterAuthentication(t *testing.T) {
	i := New(Options{RequestsPerSecond: 1})
	defer func() { _ = i.Close() }()
	requestWithToken := func(token string) int {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/api/snapshot", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		i.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if got := requestWithToken(i.token); got != http.StatusOK {
		t.Fatalf("first request status=%d", got)
	}
	if got := requestWithToken(i.token); got != http.StatusTooManyRequests {
		t.Fatalf("second request status=%d", got)
	}
	if got := requestWithToken("wrong"); got != http.StatusUnauthorized {
		t.Fatalf("unauthorized request status=%d", got)
	}
}

func TestEmbeddedAssetsAndLoopbackServerLifecycle(t *testing.T) {
	i := New(Options{})
	for _, path := range []string{"/", "/app.js", "/style.css"} {
		w := request(i, "GET", path, "")
		if w.Code != http.StatusOK || w.Body.Len() == 0 || w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("asset %s unavailable or missing headers: %d", path, w.Code)
		}
	}
	fullURL, err := i.Listen()
	mustUpdate(t, err)
	u, err := url.Parse(fullURL)
	mustUpdate(t, err)
	if u.Hostname() != "127.0.0.1" || u.Fragment == "" || u.RawQuery != "" {
		t.Fatal("server is not a loopback capability URL")
	}
	if _, err := i.Listen(); err == nil {
		t.Fatal("started a duplicate server")
	}
	u.Path, u.Fragment = "/api/snapshot", ""
	r, err := http.NewRequest("GET", u.String(), nil)
	mustUpdate(t, err)
	r.Header.Set("Authorization", "Bearer "+i.token)
	client := &http.Client{Timeout: time.Second}
	response, err := client.Do(r)
	mustUpdate(t, err)
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal(response.StatusCode)
	}
	mustUpdate(t, i.Close())
	if response, err := client.Do(r); err == nil {
		_ = response.Body.Close()
		t.Fatal("server remained reachable after Close")
	}
	if _, err := i.Listen(); err != ErrClosed {
		t.Fatalf("Listen after close = %v", err)
	}
}

func TestRemoteReadOnlyServerRejectsWrites(t *testing.T) {
	i := New(Options{})
	store := nodra.New(counterState{})
	mustAttach(t, i, "counter", store)
	fullURL, err := i.ListenRemoteReadOnly("127.0.0.1:0")
	mustUpdate(t, err)
	defer func() { _ = i.Close() }()
	u, err := url.Parse(fullURL)
	mustUpdate(t, err)
	client := &http.Client{Timeout: time.Second}
	requestURL := *u
	requestURL.Path = "/api/snapshot"
	requestURL.Fragment = ""
	get, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)
	mustUpdate(t, err)
	get.Header.Set("Authorization", "Bearer "+i.token)
	response, err := client.Do(get)
	mustUpdate(t, err)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("remote snapshot status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
	for _, path := range []string{"/api/clear", "/api/restore"} {
		requestURL.Path = path
		post, err := http.NewRequest(http.MethodPost, requestURL.String(), strings.NewReader(`{"storeId":"counter","state":{"count":7}}`))
		mustUpdate(t, err)
		post.Header.Set("Authorization", "Bearer "+i.token)
		post.Header.Set("Content-Type", "application/json")
		response, err := client.Do(post)
		mustUpdate(t, err)
		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("remote %s status = %d", path, response.StatusCode)
		}
		_ = response.Body.Close()
	}
	if store.Get().Count != 0 {
		t.Fatal("remote write changed state")
	}
}

func TestRemoteReadOnlyRequiresRestoreDisabled(t *testing.T) {
	i := New(Options{AllowRestore: true})
	defer func() { _ = i.Close() }()
	if _, err := i.ListenRemoteReadOnly("127.0.0.1:0"); err != ErrReadOnly {
		t.Fatalf("remote restore option error = %v", err)
	}
}
