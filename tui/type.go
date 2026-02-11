package tui

import "dexianta/nci/core"

type addRepoMsg struct {
	repo core.CodeRepo
	err  error
}

type loadRepoMsg struct {
	repos []core.CodeRepo
	err   error
}

type deleteRepoMsg struct {
	deleted string
	repos   []core.CodeRepo
	err     error
}

type branchesLoadRepoMsg struct {
	repos []core.CodeRepo
	err   error
}

type branchesLoadBranchConfMsg struct {
	branchConf []core.BranchConf
	err        error
}

type branchesLoadJobMsg struct {
	jobs []core.Job
	err  error
}
