package manager

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Maven struct{}

// Package-level state shared across Maven instances (All() creates new instances).
var (
	mavenCoords         = make(map[string]string) // display name → "groupId:artifactId"
	mavenPackaging      = make(map[string]string) // display name → packaging type (jar, pom, etc.)
	mavenLatestVersions = make(map[string]string) // display name → latest version
)

func (m *Maven) Name() model.Source { return model.SourceMaven }

func (m *Maven) Available() bool {
	info, err := os.Stat(mavenRepoDir())
	return err == nil && info.IsDir()
}

func (m *Maven) Scan() ([]model.Package, error) {
	repoDir := mavenRepoDir()
	prefixLen := len(repoDir) + 1 // +1 for trailing separator

	type artifact struct {
		artifactID string
		groupID    string
		version    string
		versionDir string
		modTime    int64
	}
	seen := make(map[string]*artifact) // key = artifactId

	// dirSizes tracks cumulative file sizes per directory.
	dirSizes := make(map[string]int64)

	err := filepath.WalkDir(repoDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if len(path) <= prefixLen {
			return nil
		}
		rel := filepath.ToSlash(path[prefixLen:])

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Accumulate file sizes per directory for later lookup.
		dir := filepath.Dir(path)
		dirSizes[dir] += info.Size()

		if !strings.HasSuffix(d.Name(), ".pom") {
			return nil
		}

		// Path: groupId-dirs.../artifactId/version/artifactId-version.pom
		parts := strings.Split(rel, "/")
		if len(parts) < 4 {
			return nil
		}

		version := parts[len(parts)-2]
		artifactID := parts[len(parts)-3]
		groupID := strings.Join(parts[:len(parts)-3], ".")
		key := groupID + ":" + artifactID
		modTime := info.ModTime().Unix()

		if existing, ok := seen[key]; ok {
			if modTime > existing.modTime {
				existing.version = version
				existing.versionDir = dir
				existing.modTime = modTime
			}
		} else {
			seen[key] = &artifact{
				artifactID: artifactID,
				groupID:    groupID,
				version:    version,
				versionDir: dir,
				modTime:    modTime,
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Detect artifactId collisions so we can qualify them with groupId.
	idCount := make(map[string]int)
	for _, a := range seen {
		idCount[a.artifactID]++
	}

	mavenCoords = make(map[string]string, len(seen))
	mavenPackaging = make(map[string]string, len(seen))

	pkgs := make([]model.Package, 0, len(seen))
	for _, a := range seen {
		name := a.artifactID
		if idCount[a.artifactID] > 1 {
			name = a.groupID + ":" + a.artifactID
		}
		mavenCoords[name] = a.groupID + ":" + a.artifactID
		mavenPackaging[name] = pomPackaging(filepath.Join(a.versionDir, a.artifactID+"-"+a.version+".pom"))
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     a.version,
			Source:      model.SourceMaven,
			Description: a.groupID + ":" + a.artifactID,
			InstalledAt: time.Unix(a.modTime, 0),
			SizeBytes:   dirSizes[a.versionDir],
		})
	}
	return pkgs, nil
}

var mavenHTTPClient = &http.Client{Timeout: 10 * time.Second}

func (m *Maven) CheckUpdates(pkgs []model.Package) map[string]string {
	updates := make(map[string]string)
	mavenLatestVersions = make(map[string]string)
	for _, p := range pkgs {
		// Description holds the full groupId:artifactId coordinate.
		parts := strings.SplitN(p.Description, ":", 2)
		if len(parts) != 2 {
			continue
		}
		groupID, artifactID := parts[0], parts[1]

		latest := mavenCentralLatest(groupID, artifactID)
		if latest != "" && latest != p.Version {
			updates[p.Name] = latest
			mavenLatestVersions[p.Name] = latest
		}
	}
	return updates
}

func (m *Maven) UpgradeCmd(name string) *exec.Cmd {
	coord := mavenCoords[name]
	if coord == "" {
		coord = name
	}
	version := mavenLatestVersions[name]
	if version == "" {
		version = "LATEST"
	}
	pkg := mavenPackaging[name]
	if pkg == "" {
		pkg = "jar"
	}
	// Format: groupId:artifactId:version:packaging
	artifact := coord + ":" + version + ":" + pkg
	cmd := exec.Command("mvn", "-B", "-N",
		"dependency:get",
		"-Dartifact="+artifact,
		"-Dtransitive=false",
	)
	cmd.Dir, _ = os.UserHomeDir()
	return cmd
}

// mavenCentralLatest queries Maven Central for the latest release version.
func mavenCentralLatest(groupID, artifactID string) string {
	url := fmt.Sprintf(
		"https://search.maven.org/solrsearch/select?q=g:%%22%s%%22+AND+a:%%22%s%%22&rows=1&wt=json",
		groupID, artifactID,
	)
	resp, err := mavenHTTPClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Response struct {
			Docs []struct {
				LatestVersion string `json:"latestVersion"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	if len(result.Response.Docs) == 0 {
		return ""
	}
	return result.Response.Docs[0].LatestVersion
}

// pomPackaging reads the <packaging> element from a POM file.
// Returns "jar" if not found or unreadable (jar is Maven's default).
func pomPackaging(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "jar"
	}
	s := string(data)
	start := strings.Index(s, "<packaging>")
	if start == -1 {
		return "jar"
	}
	start += len("<packaging>")
	end := strings.Index(s[start:], "</packaging>")
	if end == -1 {
		return "jar"
	}
	return strings.TrimSpace(s[start : start+end])
}

func mavenRepoDir() string {
	if dir := os.Getenv("MAVEN_REPO_LOCAL"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".m2", "repository")
}
