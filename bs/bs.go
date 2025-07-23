package main

import (
	sbbs "github.com/barbell-math/smoothbrain-bs"
)

func main() {
	sbbs.RegisterBsBuildTarget()
	sbbs.RegisterUpdateDepsTarget()
	sbbs.RegisterGoMarkDocTargets()
	sbbs.RegisterSqlcTargets("./")
	sbbs.RegisterCommonGoCmdTargets(sbbs.NewGoTargets().
		DefaultFmtTarget().
		DefaultGenerateTarget().
		DefaultTestTarget(),
	)
	sbbs.RegisterMergegateTarget(sbbs.MergegateTargets{
		CheckDepsUpdated:     true,
		CheckReadmeGomarkdoc: true,
		FmtTarget:            sbbs.DefaultFmtTargetName,
		TestTarget:           sbbs.DefaultTestTargetName,
		GenerateTarget:       sbbs.DefaultGenerateTargetName,
		PreStages:            []sbbs.StageFunc{sbbs.TargetAsStage("sqlcInstall")},
	})

	sbbs.Main("bs")
}
