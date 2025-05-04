package main

import (
	"context"

	sbbs "github.com/barbell-math/smoothbrain-bs"
)

func main() {
	sbbs.RegisterBsBuildTarget()
	sbbs.RegisterUpdateDepsTarget()
	sbbs.RegisterGoMarkDocTargets()
	sbbs.RegisterSqlcTargets("./")
	sbbs.RegisterCommonGoCmdTargets(sbbs.GoTargets{
		GenericTestTarget:     true,
		GenericBenchTarget:    true,
		GenericFmtTarget:      true,
		GenericGenerateTarget: true,
	})

	sbbs.RegisterTarget(
		context.Background(),
		"mergegate",
		sbbs.TargetAsStage("fmt"),
		sbbs.GitDiffStage("Fix formatting to get a passing run!", "fmt"),
		sbbs.TargetAsStage("gomarkdocInstall"),
		sbbs.TargetAsStage("gomarkdocReadme"),
		sbbs.GitDiffStage("Readme is out of date", "gomarkdocReadme"),
		sbbs.TargetAsStage("updateDeps"),
		sbbs.GitDiffStage("Out of date packages were detected", "updateDeps"),
		sbbs.TargetAsStage("sqlcInstall"),
		sbbs.TargetAsStage("generate"),
		sbbs.TargetAsStage("test"),
	)

	sbbs.Main("bs")
}
