package main

import (
	"fmt"

	"github.com/damc-dev/svnpath"
)

func main() {
	svnUrl := "<svn url>"
	svnInfo, err := svnpath.SvnStat(svnUrl)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", svnInfo)
}
