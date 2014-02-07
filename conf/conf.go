package conf

type PolicyCmd string

const (
	ALLOW   PolicyCmd = "allow"
	DENY              = "deny"
	MIN_FEE           = "min-fee"
	// TODO(ortutay): add rate-limit
	// TODO(ortutay): additional policy commands
)

// TODO(ortutay): implement real selectors; PolicySelector is just a placeholder
type PolicySelector string

const (
	GLOBAL PolicySelector = "global"
)

type Policy struct {
	Selector PolicySelector
	Cmd      PolicyCmd
	Args     []interface{}
}
