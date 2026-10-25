// Package source loads explicit configuration sources without running repository code.
package source

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxSize = 64 << 20

type Loader struct{ Client *http.Client }

func (l Loader) Open(ctx context.Context, location, digest string) (string, func(), error) {
	if !strings.Contains(location, "://") {
		p, err := filepath.Abs(location)
		if err != nil {
			return "", nil, err
		}
		if _, err = catalog.Digest(p); err != nil {
			return "", nil, err
		}
		return p, func() {}, nil
	}
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return "", nil, fmt.Errorf("remote profile source must be an HTTPS URL")
	}
	want, err := hex.DecodeString(digest)
	if err != nil || len(want) != 32 {
		return "", nil, fmt.Errorf("remote profile bundles require --sha256 with 64 hexadecimal characters")
	}
	client := l.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) > 5 {
			return fmt.Errorf("unsafe or excessive profile download redirects")
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := copyClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", nil, fmt.Errorf("profile download returned HTTP %d", resp.StatusCode)
	}
	root, err := os.MkdirTemp("", "outfit-profiles-")
	if err != nil {
		return "", nil, err
	}
	clean := func() { os.RemoveAll(root) }
	fail := func(err error) (string, func(), error) { clean(); return "", nil, err }
	archive, err := os.CreateTemp(root, "download-")
	if err != nil {
		return fail(err)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(archive, h), io.LimitReader(resp.Body, maxSize+1))
	archive.Close()
	if err != nil {
		return fail(err)
	}
	if n > maxSize {
		return fail(fmt.Errorf("profile bundle exceeds 64 MiB"))
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), digest) {
		return fail(fmt.Errorf("profile bundle SHA-256 mismatch"))
	}
	f, err := os.Open(archive.Name())
	if err != nil {
		return fail(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return fail(err)
	}
	defer z.Close()
	r := tar.NewReader(io.LimitReader(z, maxSize+1))
	bundle := filepath.Join(root, "bundle")
	os.Mkdir(bundle, 0700)
	var total int64
	seen := map[string]bool{}
	for count := 0; ; count++ {
		head, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fail(err)
		}
		if count > 5000 {
			return fail(fmt.Errorf("too many bundle entries"))
		}
		if !catalog.SafeRelative(head.Name) {
			return fail(fmt.Errorf("unsafe archive path %q", head.Name))
		}
		p := filepath.Join(bundle, filepath.FromSlash(head.Name))
		if seen[p] {
			return fail(fmt.Errorf("duplicate archive entry %s", head.Name))
		}
		seen[p] = true
		switch head.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(p, 0700)
		case tar.TypeReg, tar.TypeRegA:
			total += head.Size
			if head.Size < 0 || total > maxSize {
				return fail(fmt.Errorf("expanded bundle exceeds 64 MiB"))
			}
			err = os.MkdirAll(filepath.Dir(p), 0700)
			if err == nil {
				var out *os.File
				out, err = os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err == nil {
					_, err = io.Copy(out, r)
					out.Close()
				}
			}
		default:
			return fail(fmt.Errorf("unsupported archive entry type for %s", head.Name))
		}
		if err != nil {
			return fail(err)
		}
	}
	if _, err = os.Stat(filepath.Join(bundle, "outfit.yaml")); err != nil {
		return fail(fmt.Errorf("bundle must contain outfit.yaml at its root"))
	}
	return bundle, clean, nil
}
