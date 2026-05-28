package backlog

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The derived view directories and aggregated overview files all
// live under `.am/` so the project root stays scannable. Only
// human-edited content (stories, per-backlog overview, backlog
// folders) is visible at the root. `.am/archive/` is NOT used:
// archived stories stay alongside their owning backlog at
// `<backlog>/archive/` so a story's history is local to the
// backlog that produced it.
const (
	configFileName        = ".am/config.yaml"
	indexFileName         = "index.md"
	velocityFileName      = ".am/velocity.md"
	velocityDirectoryName = ".am/velocity"
	archiveDirectoryName  = "archive"
	TagsDirectoryName     = ".am/tags"
	TagsFileName          = ".am/tags.md"
	usersDirectoryName    = ".am/users"
	usersFileName         = ".am/users.md"
	timelineFileName      = ".am/timeline.md"
	timelineDirectoryName = ".am/timeline"
)

var (
	ForbiddenBacklogNames = []string{velocityDirectoryName, archiveDirectoryName, TagsDirectoryName, usersDirectoryName, timelineDirectoryName}
	ForbiddenItemNames    = []string{archiveDirectoryName, "_priority", "_icebox"}
)

type BacklogsStructure struct {
	root string
}

func NewBacklogsStructure(root string) *BacklogsStructure {
	return &BacklogsStructure{root: root}
}

func (s *BacklogsStructure) Root() string {
	return s.root
}

func (s *BacklogsStructure) ConfigFile() string {
	return filepath.Join(s.root, configFileName)
}

func (s *BacklogsStructure) IndexFile() string {
	return filepath.Join(s.root, indexFileName)
}

func (s *BacklogsStructure) VelocityFile() string {
	return filepath.Join(s.root, velocityFileName)
}

func (s *BacklogsStructure) TagsFile() string {
	return filepath.Join(s.root, TagsFileName)
}

func (s *BacklogsStructure) UsersFile() string {
	return filepath.Join(s.root, usersFileName)
}

func (s *BacklogsStructure) TimelineFile() string {
	return filepath.Join(s.root, timelineFileName)
}

func (s *BacklogsStructure) UsersDirectory() string {
	return filepath.Join(s.root, usersDirectoryName)
}

func (s *BacklogsStructure) TagsDirectory() string {
	return filepath.Join(s.root, TagsDirectoryName)
}

func (s *BacklogsStructure) TimelineDirectory() string {
	return filepath.Join(s.root, timelineDirectoryName)
}

// BacklogDirs returns the project's backlog directories. Convention:
// a backlog is a folder at the project root that has a matching
// `<dir-name>.md` overview file alongside it. The canonical name is
// `stories` (one project = one backlog); legacy `am-*` folders from
// earlier projects still validate as long as the overview is there.
// Other content (docs/, confluence/, etc.) is silently ignored.
func (s *BacklogsStructure) BacklogDirs() ([]string, error) {
	infos, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") || IsForbiddenBacklogName(info.Name()) {
			continue
		}
		dir := filepath.Join(s.root, info.Name())
		if _, ok := FindOverviewFileInRootDirectory(dir); !ok {
			continue
		}
		result = append(result, dir)
	}
	sort.Strings(result)
	return result, nil
}
