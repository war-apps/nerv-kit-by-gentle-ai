// Package selfupgradetest fakes a GitHub releases API and its download
// server for one binary, so selfupgrade.Client and its consumers can be
// tested without a real network call.
package selfupgradetest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Release is one GitHub release the fake server publishes.
type Release struct {
	Tag        string
	Prerelease bool
	Draft      bool
}

// Server is the fake GitHub releases API and download server for Binary.
type Server struct {
	// Binary is the name of the binary every archive this server serves
	// contains (plain on linux/darwin, "<Binary>.exe" inside the windows
	// .zip).
	Binary string
	// Releases is the list this server's releases endpoints report,
	// newest first, as GitHub itself returns them.
	Releases []Release
	// Corrupt makes checksums.txt list a wrong sum for every archive,
	// simulating a corrupted download.
	Corrupt bool
	// Downloads records every "<tag>/<file>" this server has served.
	Downloads []string

	t   *testing.T
	srv *httptest.Server
}

// New starts a Server for binary, closed automatically at the end of t.
func New(t *testing.T, binary string, releases ...Release) *Server {
	t.Helper()
	s := &Server{t: t, Binary: binary, Releases: releases}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

// Client is the http.Client that trusts this server's TLS, when any.
func (s *Server) Client() *http.Client { return s.srv.Client() }

// APIBase is this server's releases API base, for selfupgrade.Client.APIBase.
func (s *Server) APIBase() string { return s.srv.URL + "/api" }

// DownloadBase is this server's download base, for
// selfupgrade.Client.DownloadBase.
func (s *Server) DownloadBase() string { return s.srv.URL + "/download" }

// BinaryBody is the content of Binary inside the archive for tag.
func (s *Server) BinaryBody(tag string) string {
	return "#!new " + s.Binary + " " + tag + "\n"
}

func (s *Server) archiveName(tag, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return s.Binary + "_" + strings.TrimPrefix(tag, "v") + "_" + goos + "_" + goarch + ext
}

// TarGz builds the tar.gz release archive of tag, holding Binary.
func (s *Server) TarGz(tag string) []byte {
	s.t.Helper()
	body := s.BinaryBody(tag)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: s.Binary, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		s.t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		s.t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		s.t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		s.t.Fatal(err)
	}
	return buf.Bytes()
}

// Zip builds the windows release archive of tag, holding "<Binary>.exe".
func (s *Server) Zip(tag string) []byte {
	s.t.Helper()
	body := s.BinaryBody(tag)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(s.Binary + ".exe")
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := f.Write([]byte(body)); err != nil {
		s.t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		s.t.Fatal(err)
	}
	return buf.Bytes()
}

func (s *Server) serve(rw http.ResponseWriter, r *http.Request) {
	type apiRelease struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	switch {
	case r.URL.Path == "/api/releases/latest":
		for _, rel := range s.Releases {
			if !rel.Prerelease && !rel.Draft {
				_ = json.NewEncoder(rw).Encode(apiRelease{TagName: rel.Tag})
				return
			}
		}
		http.NotFound(rw, r)
	case r.URL.Path == "/api/releases" && r.URL.Query().Get("per_page") == "50":
		list := []apiRelease{}
		for _, rel := range s.Releases {
			list = append(list, apiRelease{TagName: rel.Tag, Prerelease: rel.Prerelease, Draft: rel.Draft})
		}
		_ = json.NewEncoder(rw).Encode(list)
	case strings.HasPrefix(r.URL.Path, "/download/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/"), "/")
		if len(parts) != 2 {
			http.NotFound(rw, r)
			return
		}
		tag, file := parts[0], parts[1]
		s.Downloads = append(s.Downloads, tag+"/"+file)
		switch {
		case file == s.archiveName(tag, "linux", "amd64"), file == s.archiveName(tag, "darwin", "arm64"):
			_, _ = rw.Write(s.TarGz(tag))
		case file == s.archiveName(tag, "windows", "amd64"):
			_, _ = rw.Write(s.Zip(tag))
		case file == "checksums.txt":
			linuxSum := sha256.Sum256(s.TarGz(tag))
			linuxHex := hex.EncodeToString(linuxSum[:])
			windowsSum := sha256.Sum256(s.Zip(tag))
			windowsHex := hex.EncodeToString(windowsSum[:])
			if s.Corrupt {
				linuxHex = strings.Repeat("0", 64)
				windowsHex = strings.Repeat("0", 64)
			}
			fmt.Fprintf(rw, "%s  %s\n%s  %s\n", linuxHex, s.archiveName(tag, "linux", "amd64"), windowsHex, s.archiveName(tag, "windows", "amd64"))
		default:
			http.NotFound(rw, r)
		}
	default:
		http.NotFound(rw, r)
	}
}
