package svnpath

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var (
	repoName = "TestRepo"
	repoURL  = "https://<server>:<port>/svn/" + repoName
)

const svnInfoRunResult = `
Path: TestRepo
URL: https://<server>:<port>/svn/TestRepo
Repository Root: https://<server>:<port>/svn/TestRepo
Repository UUID: 5e88e5b6-f55b-11de-b280-d7f93031a90f
Revision: 17954
Node Kind: directory
Last Changed Author: david
Last Changed Rev: 17954
Last Changed Date: 2017-10-24 15:20:01 -0500 (Tue, 24 Oct 2017)`

func TestSvnInfoHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprintf(os.Stdout, svnInfoRunResult)
	os.Exit(0)
}

func fakeExecCommandContextInfo(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSvnInfoHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestSvnStat(t *testing.T) {
	execCommandContext = fakeExecCommandContextInfo

	svnInfo, err := SvnStat(repoURL)
	if err != nil {
		t.Error(err)
	}

	if svnInfo.Name() != repoName {
		t.Errorf("Expected Name() to return %s, but was %s", repoName, svnInfo.Name())
	}

	if !svnInfo.IsDir() {
		t.Error("Expected IsDir() to return true")
	}

	t.Logf("%+v", svnInfo)
	dirs, _ := svnInfo.Dirs()
	t.Logf("%+v", dirs)
}

const svnLsRunResult = `trunk/
tags/
branches/`

func TestSvnLsHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprintf(os.Stdout, svnLsRunResult)
	os.Exit(0)
}

func fakeExecCommmandContextSvnLs(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSvnLsHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}
func TestSvnDirs(t *testing.T) {

	execCommandContext = fakeExecCommandContextInfo
	svnInfo, err := SvnStat(repoURL)
	execCommandContext = fakeExecCommmandContextSvnLs

	if err != nil {
		t.Error(err)
	}

	if svnInfo.Name() != repoName {
		t.Errorf("Expected Name() to return %s, but was %s", repoName, svnInfo.Name())
	}

	if !svnInfo.IsDir() {
		t.Error("Expected IsDir() to return true")
	}

	dirs, err := svnInfo.Dirs()
	if err != nil {
		t.Error("Error while executing Dirs()")
	}

	if len(dirs) <= 0 {
		t.Error("Dirs() returned no sub directories")
	}

	t.Logf("%+v", svnInfo)
}

const svnStatWhereAccessForbiddenRunResult = `svn: E175013: Unable to connect to a repository at URL 'https://<server>:<port>/svn/TestRepo'
svn: E175013: Access to 'https://<server>:<port>/svn/TestRepo' forbidden`

func TestSvnStatWhereAccessForbiddenHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprintf(os.Stderr, svnStatWhereAccessForbiddenRunResult)
	os.Exit(1)
}

func fakeExecCommmandContextAccessForbidden(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestSvnStatWhereAccessForbiddenHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestSvnStatWhereAccessForbidden(t *testing.T) {
	execCommandContext = fakeExecCommmandContextAccessForbidden

	svnInfo, err := SvnStat(repoURL)

	if err == nil {
		t.Error("Expected error ErrAccessForbidden")
	}
	if err != nil && err != ErrAccessForbidden {
		t.Error(err)
	}

	t.Logf("%+v", svnInfo)
}

func TestWalkWhereAccessForbidden(t *testing.T) {
	execCommandContext = fakeExecCommmandContextAccessForbidden

	walkFn := func(urlPath string, info SvnObject, err error) error {
		if err == ErrAccessForbidden {
			t.Logf(" ## ACESS FORBIDDEN: %s", urlPath)
		} else if err != nil {
			return err
		}
		return nil
	}
	err := Walk(repoURL, walkFn)
	if err != nil {
		t.Errorf("Error returned from Walk function")
	}
}
