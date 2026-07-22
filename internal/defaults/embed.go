package defaults

import (_ "embed")

//go:embed sync.yml
var SyncFile string

//go:embed config.yml
var ConfigFile string