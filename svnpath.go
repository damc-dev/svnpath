package svnpath

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var (
	SkipDir            = errors.New("skip this directory")
	ErrNotDirectory    = errors.New("sub directories can't be found for non directory node type")
	ErrAccessForbidden = errors.New("access to given svn url is forbidden")
	SvnStat            = svnStat
	execCommandContext = exec.CommandContext
)

type WalkFunc func(urlPath string, info SvnObject, err error) error

func Walk(root string, walkFn WalkFunc) error {
	info, err := SvnStat(root)
	if err != nil {
		err = walkFn(root, nil, err)
	} else {
		err = walk(root, info, walkFn)
	}
	if err == SkipDir {
		return nil
	}
	return err
}

// walk recursively descends path, calling walkFn.
func walk(path string, info SvnObject, walkFn WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}

	names, err := readDirNames(path)
	err1 := walkFn(path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}

	for _, name := range names {
		filename := Join(path, name)
		fileInfo, err := SvnStat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != SkipDir {
				return err
			}
		} else {
			err = walk(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != SkipDir {
					return err
				}
			}
		}
	}
	return nil

}

type SvnObject interface {
	Name() string
	IsDir() bool
	Dirs() ([]string, error)
}
type SvnInfo struct {
	url                string
	name               string
	nodeKind           string
	revision           string
	lastChangeAuthor   string
	lastChangeRevision string
	lastChangeDate     string
}

func (i *SvnInfo) Name() string {
	return i.name
}

func (i *SvnInfo) IsDir() bool {
	return i.nodeKind == "directory"
}

func (i *SvnInfo) Dirs() ([]string, error) {
	if !i.IsDir() {
		return nil, ErrNotDirectory
	}
	return readDirNames(i.url)
}

func svnStat(url string) (SvnObject, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmdName := "svn"
	cmdArgs := []string{"info", Clean(url)}

	cmd := execCommandContext(ctx, cmdName, cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to open stdout pipe for svn info command")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cmd.Start()
	info := SvnInfo{}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				kv := strings.Split(line, ": ")
				switch kv[0] {
				case "Path":
					info.name = kv[1]
				case "URL":
					info.url = kv[1]
				case "Revision":
					info.revision = kv[1]
				case "Node Kind":
					info.nodeKind = kv[1]
				case "Last Changed Author":
					info.lastChangeAuthor = kv[1]
				case "Last Changed Rev":
					info.lastChangeRevision = kv[1]
				case "Last Changed Date":
					info.lastChangeDate = kv[1]
				}
			}
		}
	}()

	err = cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("svn info command timed out")
	}

	if err != nil {
		if strings.Contains(stderr.String(), "E175013") {
			return nil, ErrAccessForbidden
		}
		fmt.Printf("Command failed with error: %s\n\tstderr: %s", err, stderr.String())
		return nil, err
	}

	return &info, nil
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func Join(elem ...string) string {
	for i, e := range elem {
		if e != "" {
			return strings.Join(elem[i:], "/")
		}
	}
	return ""
}

func Clean(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}

// readDirNames reads the svn url directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(url string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmdName := "svn"
	cmdArgs := []string{"ls", url}

	cmd := execCommandContext(ctx, cmdName, cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to open stdout pipe for svn info command")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	cmd.Start()

	files := []string{}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				files = append(files, strings.Trim(line, "/"))
			}
		}
	}()

	err = cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("svn ls command timed out")
	}

	if err != nil {
		if strings.Contains(stderr.String(), "E175013") {
			return nil, ErrAccessForbidden
		}
		fmt.Printf("Command failed with error: %s\n\tstderr: %s", err, stderr.String())
		return nil, err
	}

	return files, nil
}
