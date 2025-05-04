package main

import (
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
	sbbs.RegisterMergegateTarget(sbbs.MergegateTargets{
		CheckDepsUpdated:     true,
		CheckReadmeGomarkdoc: true,
		CheckFmt:             true,
		CheckUnitTests:       true,
	})

	sbbs.Main("bs")
}
