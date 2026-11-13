package selfupgrade

import (
	"context"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade/selfupgradetest"
)

// TestCheckPlatform checks every goos/goarch combination a typical release
// pipeline builds (linux, darwin and windows, each amd64/arm64) is
// accepted, and a platform or architecture without a prebuilt binary is
// refused with the caller-supplied install hint.
func TestCheckPlatform(t *testing.T) {
	tests := []struct {
		goos, goarch string
		wantErr      bool
	}{
		{"linux", "amd64", false},
		{"linux", "arm64", false},
		{"darwin", "amd64", false},
		{"darwin", "arm64", false},
		{"windows", "amd64", false},
		{"windows", "arm64", false},
		{"freebsd", "amd64", true},
		{"windows", "386", true},
		{"linux", "386", true},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			err := CheckPlatform(tt.goos, tt.goarch, "kit", "install with: go install example.com/kit@latest")
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPlatform(%q, %q) = %v, want error: %v", tt.goos, tt.goarch, err, tt.wantErr)
			}
		})
	}
}

func TestCheckPlatform_MessageNamesBinaryAndHint(t *testing.T) {
	err := CheckPlatform("freebsd", "amd64", "kit", "install with: go install example.com/kit@latest")
	if err == nil {
		t.Fatal("want an error")
	}
	want := "Unsupported OS: freebsd. Only Linux, macOS and Windows have prebuilt kit binaries.\ninstall with: go install example.com/kit@latest"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if _, ok := err.(*PlatformError); !ok {
		t.Errorf("error type = %T, want *PlatformError", err)
	}
}

// TestArchiveName checks the archive extension: .tar.gz everywhere except
// windows, which is packaged as .zip.
func TestArchiveName(t *testing.T) {
	tests := []struct {
		binary, tag, goos, goarch, want string
	}{
		{"kit", "v1.2.3", "linux", "amd64", "kit_1.2.3_linux_amd64.tar.gz"},
		{"kit", "v1.2.3", "darwin", "arm64", "kit_1.2.3_darwin_arm64.tar.gz"},
		{"kit", "v1.2.3", "windows", "amd64", "kit_1.2.3_windows_amd64.zip"},
		{"kit", "v1.2.3", "windows", "arm64", "kit_1.2.3_windows_arm64.zip"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			if got := ArchiveName(tt.binary, tt.tag, tt.goos, tt.goarch); got != tt.want {
				t.Errorf("ArchiveName(%q, %q, %q, %q) = %q, want %q", tt.binary, tt.tag, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		release, current string
		want             bool
	}{
		{"v5.1.0", "v5.0.9", true},
		{"v5.0.9", "v5.1.0", false},
		{"v5.1.0", "v5.1.0", false},
		{"v5.1.0", "dev", true},
		{"v5.1.0-rc.1", "v5.1.0", false},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.release, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.release, tt.current, got, tt.want)
		}
	}
}

// --- Replace: tar.gz / zip extraction and placement ---

func TestReplace_NonWindowsRenamesOver(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			exe := dir + "/kit"
			writeFile(t, exe, []byte("old"))
			archive := simpleTarGz(t, "kit", "new")

			if err := Replace(archive, exe, goos, "kit"); err != nil {
				t.Fatalf("Replace: %v", err)
			}
			got := readFile(t, exe)
			if got != "new" {
				t.Errorf("exe = %q, want %q", got, "new")
			}
			if exists(exe + ".old") {
				t.Errorf("exe.old exists on %s, want none", goos)
			}
		})
	}
}

func TestReplace_WindowsExtractsZipAndPlacesAsideOld(t *testing.T) {
	dir := t.TempDir()
	exe := dir + "/kit.exe"
	writeFile(t, exe, []byte("old"))
	archive := simpleZip(t, "kit", "new")

	if err := Replace(archive, exe, "windows", "kit"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	got := readFile(t, exe)
	if got != "new" {
		t.Errorf("exe = %q, want %q", got, "new")
	}
	old := readFile(t, exe+".old")
	if old != "old" {
		t.Errorf("exe.old = %q, want the original %q", old, "old")
	}
}

func TestReplace_ZipMissingBinaryEntry(t *testing.T) {
	archive := simpleZipNamed(t, "README.md", "readme")
	exe := t.TempDir() + "/kit.exe"
	writeFile(t, exe, []byte("old"))

	err := Replace(archive, exe, "windows", "kit")
	if err == nil {
		t.Fatal("want an error when kit.exe is missing from the archive")
	}
	if readFile(t, exe) != "old" {
		t.Error("exe was modified despite the missing entry")
	}
}

func TestPlaceWindows_RestoresOriginalWhenPlacementFails(t *testing.T) {
	dir := t.TempDir()
	exe := dir + "/kit.exe"
	writeFile(t, exe, []byte("old"))
	missingTmp := dir + "/does-not-exist"

	if err := placeWindows(missingTmp, exe); err == nil {
		t.Fatal("want an error when placing the new binary fails")
	}
	if readFile(t, exe) != "old" {
		t.Error("exe missing or changed after a failed placement")
	}
	if exists(exe + ".old") {
		t.Error("exe.old left behind after a restore")
	}
}

func TestPlaceWindows_RemovesStaleOldFirst(t *testing.T) {
	dir := t.TempDir()
	exe := dir + "/kit.exe"
	tmp := dir + "/new"
	writeFile(t, exe, []byte("current"))
	writeFile(t, exe+".old", []byte("stale"))
	writeFile(t, tmp, []byte("new"))

	if err := placeWindows(tmp, exe); err != nil {
		t.Fatalf("placeWindows: %v", err)
	}
	if readFile(t, exe) != "new" {
		t.Errorf("exe = %q, want %q", readFile(t, exe), "new")
	}
	if readFile(t, exe+".old") != "current" {
		t.Errorf("exe.old = %q, want the just-replaced %q", readFile(t, exe+".old"), "current")
	}
}

func TestCleanupStaleReplacement_RemovesLeftoverOld(t *testing.T) {
	dir := t.TempDir()
	exe := dir + "/kit.exe"
	old := exe + ".old"
	writeFile(t, old, []byte("stale"))
	CleanupStaleReplacement(exe)
	if exists(old) {
		t.Error("exe.old still present after cleanup")
	}
}

func TestCleanupStaleReplacement_NoOldIsHarmless(t *testing.T) {
	exe := t.TempDir() + "/kit"
	CleanupStaleReplacement(exe) // must not panic
}

// --- Client.Resolve / Download / VerifyChecksum ---

func TestClient_Resolve_Stable(t *testing.T) {
	s := selfupgradetest.New(t, "kit",
		selfupgradetest.Release{Tag: "v5.2.0-rc.1", Prerelease: true},
		selfupgradetest.Release{Tag: "v5.1.0"},
		selfupgradetest.Release{Tag: "v5.0.0"},
	)
	c := Client{HTTP: s.Client(), APIBase: s.APIBase(), DownloadBase: s.DownloadBase(), BinaryName: "kit"}

	got, err := c.Resolve(context.Background(), Stable)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != "v5.1.0" {
		t.Errorf("Resolve(Stable) = %q, want %q", got, "v5.1.0")
	}
}

func TestClient_Resolve_PrereleaseChannels(t *testing.T) {
	releases := []selfupgradetest.Release{
		{Tag: "v5.2.0-rc.1", Prerelease: true},
		{Tag: "v5.2.0-alpha.3", Prerelease: true},
		{Tag: "v5.1.0"},
	}
	s := selfupgradetest.New(t, "kit", releases...)
	c := Client{HTTP: s.Client(), APIBase: s.APIBase(), DownloadBase: s.DownloadBase(), BinaryName: "kit"}

	if got, err := c.Resolve(context.Background(), RC); err != nil || got != "v5.2.0-rc.1" {
		t.Errorf("Resolve(RC) = %q, %v, want %q", got, err, "v5.2.0-rc.1")
	}
	if got, err := c.Resolve(context.Background(), Alpha); err != nil || got != "v5.2.0-alpha.3" {
		t.Errorf("Resolve(Alpha) = %q, %v, want %q", got, err, "v5.2.0-alpha.3")
	}
}

func TestClient_Resolve_NoPrereleaseOfChannel(t *testing.T) {
	s := selfupgradetest.New(t, "kit", selfupgradetest.Release{Tag: "v5.2.0-rc.1", Prerelease: true})
	c := Client{HTTP: s.Client(), APIBase: s.APIBase(), DownloadBase: s.DownloadBase(), BinaryName: "kit"}

	if _, err := c.Resolve(context.Background(), Alpha); err == nil {
		t.Fatal("want an error when no alpha release is published")
	}
}

func TestClient_Download_VerifiesChecksum(t *testing.T) {
	s := selfupgradetest.New(t, "kit", selfupgradetest.Release{Tag: "v5.1.0"})
	c := Client{HTTP: s.Client(), APIBase: s.APIBase(), DownloadBase: s.DownloadBase(), BinaryName: "kit"}

	archive, err := c.Download(context.Background(), "v5.1.0", "linux", "amd64", t.TempDir())
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if readFile(t, archive) == "" {
		t.Error("downloaded archive is empty")
	}
}

func TestClient_Download_CorruptChecksumFails(t *testing.T) {
	s := selfupgradetest.New(t, "kit", selfupgradetest.Release{Tag: "v5.1.0"})
	s.Corrupt = true
	c := Client{HTTP: s.Client(), APIBase: s.APIBase(), DownloadBase: s.DownloadBase(), BinaryName: "kit"}

	if _, err := c.Download(context.Background(), "v5.1.0", "linux", "amd64", t.TempDir()); err == nil {
		t.Fatal("want a checksum verification error")
	}
}
