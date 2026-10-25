package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func archive(t *testing.T, name string, kind byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	w := tar.NewWriter(z)
	body := []byte("schema: 1\nname: test\n")
	h := &tar.Header{Name: name, Mode: 0600, Typeflag: kind, Size: int64(len(body))}
	if kind == tar.TypeSymlink {
		h.Size = 0
		h.Linkname = "/etc/passwd"
	}
	w.WriteHeader(h)
	if h.Size > 0 {
		w.Write(body)
	}
	w.Close()
	z.Close()
	return b.Bytes()
}
func TestVerifiedBundle(t *testing.T) {
	data := archive(t, "outfit.yaml", tar.TypeReg)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()
	loader := Loader{Client: srv.Client()}
	root, cleanup, err := loader.Open(t.Context(), srv.URL, fmt.Sprintf("%x", sha256.Sum256(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(root, "outfit.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loader.Open(t.Context(), srv.URL, "deadbeef"); err == nil {
		t.Fatal("bad digest accepted")
	}
}
func TestRejectArchiveTraversal(t *testing.T) {
	for _, p := range []struct {
		name string
		kind byte
	}{{"../escaped", tar.TypeReg}, {"/absolute", tar.TypeReg}, {"link", tar.TypeSymlink}} {
		data := archive(t, p.name, p.kind)
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
		_, cleanup, err := (Loader{Client: srv.Client()}).Open(t.Context(), srv.URL, fmt.Sprintf("%x", sha256.Sum256(data)))
		if cleanup != nil {
			cleanup()
		}
		srv.Close()
		if err == nil {
			t.Fatalf("accepted %s", p.name)
		}
	}
}
func TestHTTPSRequired(t *testing.T) {
	if _, _, err := (Loader{}).Open(t.Context(), "http://example.com/profile.tar.gz", ""); err == nil {
		t.Fatal("accepted plaintext network source")
	}
}
