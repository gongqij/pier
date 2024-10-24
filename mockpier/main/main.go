package main

import (
	"flag"
	"fmt"
	"github.com/meshplus/pier/internal/repo"
	"github.com/meshplus/pier/mockpier"
	"time"
)

var repoStr = flag.String("repo", ".", "specify repo path")

func main() {
	repoRoot, err := repo.PathRootWithDefault(*repoStr)
	if err != nil {
		fmt.Println("step1 error: " + err.Error())
		return
	}

	mockPier, err := mockpier.NewMockPier(repoRoot)
	if err != nil {
		fmt.Println("step2 error: " + err.Error())
		return
	}

	err = mockPier.Start()
	if err != nil {
		fmt.Println("step3 error: " + err.Error())
		return
	}
	time.Sleep(time.Hour)
	return
}
