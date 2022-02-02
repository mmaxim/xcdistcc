package client

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"mmaxim.org/xcdistcc/common"
)

type IncludeFinder struct {
	*common.LabelLogger
	directoryListCache map[string]map[string]bool
}

func NewIncludeFinder(logger common.Logger) *IncludeFinder {
	return &IncludeFinder{
		LabelLogger:        common.NewLabelLogger("IncludeFinder", logger),
		directoryListCache: make(map[string]map[string]bool),
	}
}

func (f *IncludeFinder) listDirectory(dir string) (map[string]bool, error) {
	dirlist, ok := f.directoryListCache[dir]
	if ok {
		return dirlist, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	ret := make(map[string]bool, len(names))
	for _, name := range names {
		ret[name] = true
	}
	f.directoryListCache[dir] = ret
	return ret, nil
}

func (f *IncludeFinder) includeFromLine(origline string) (include string, ok bool) {
	line := strings.TrimSpace(origline)
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	line = line[1:]
	toks := strings.Fields(line)
	if len(toks) < 2 {
		return "", false
	}
	if toks[0] != "include" && toks[0] != "import" {
		return "", false
	}
	path := toks[1]
	f.Debug("includeFromLine: origline: %s path: %s", origline, path)
	return path[1 : len(path)-1], true
}

func (f *IncludeFinder) getIncludesFromFile(path string) (res []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		include, ok := f.includeFromLine(scanner.Text())
		if !ok {
			continue
		}
		res = append(res, include)
	}
	return res, nil
}

func (f *IncludeFinder) locateInclude(relpath string, includeDirs []string) (string, error) {
	for _, includeDir := range includeDirs {
		abspath := filepath.Join(includeDir, relpath)
		if _, err := os.Stat(abspath); err != nil {
			continue
		}
		return abspath, nil
	}
	return "", fmt.Errorf("unable to find file: %s", relpath)
}

func (f *IncludeFinder) collectIncludes(path string, includeDirs []string,
	res map[string]common.IncludeData) {
	f.Debug("collecting: %s", path)

	includes, err := f.getIncludesFromFile(path)
	if err != nil {
		f.Debug("collectIncludes: failed to get includes from: path: %s err: %s", path, err)
		return
	}
	allIncludeDirs := append([]string{filepath.Dir(path)}, includeDirs...)
	for _, include := range includes {
		abspath, err := f.locateInclude(include, allIncludeDirs)
		if err != nil {
			f.Debug("failed to locate include: %s err: %s", include, err)
			continue
		}
		if _, ok := res[abspath]; ok {
			continue
		}

		dat, err := os.ReadFile(abspath)
		if err != nil {
			f.Debug("failed to read included file: abspath: %s err: %s", abspath, err)
			continue
		}
		res[abspath] = common.IncludeData{
			Path: abspath,
			Data: string(dat[:]),
		}
		f.collectIncludes(abspath, includeDirs, res)
	}
}

func (f *IncludeFinder) loadForcedIncludes(res map[string]common.IncludeData, includeDirs []string) {
	forced := []string{"/Users/mike/go/src/git.zoom.us/keybase/Vendors/boost_1_72_0/boost/preprocessor/iteration/detail/iter/forward1.hpp"}
	for _, force := range forced {
		if _, ok := res[force]; ok {
			continue
		}
		dat, err := os.ReadFile(force)
		if err != nil {
			f.Debug("failed to read included file: abspath: %s err: %s", force, err)
			continue
		}
		res[force] = common.IncludeData{
			Path: force,
			Data: string(dat[:]),
		}
		f.collectIncludes(force, includeDirs, res)
	}
}

func (f *IncludeFinder) Preprocess(cmd *common.XcodeCmd) (code []byte, retcmd *common.XcodeCmd, res []common.IncludeData, err error) {
	retcmd = cmd.Clone()
	dirs := cmd.IncludeDirs()
	for _, dir := range dirs {
		f.Debug("include dir: %s", dir)
	}
	inputPath, err := cmd.GetInputFilepath()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get input path")
	}
	if code, err = os.ReadFile(inputPath); err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to read input file")
	}

	mres := make(map[string]common.IncludeData)
	f.loadForcedIncludes(mres, dirs)
	f.collectIncludes(inputPath, dirs, mres)
	res = make([]common.IncludeData, 0, len(mres))
	for _, id := range mres {
		res = append(res, id)
		f.Debug("include: %s", id.Path)
	}
	retcmd.PushIncludeDirBack(filepath.Dir(inputPath))
	return code, retcmd, res, nil
}
