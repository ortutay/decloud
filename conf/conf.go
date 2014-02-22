package conf

import (
	"fmt"
)

type BtcAddr string

type PolicyCmd string

const (
	ALLOW   PolicyCmd = "allow"
	DENY              = "deny"
	MIN_FEE           = "min-fee"
	MIN_COINS = "min-coins"
	MAX_WORK = "max-work"
	// TODO(ortutay): add rate-limit
	// TODO(ortutay): additional policy commands
)

type PolicySelector struct {
	Service string
	Method  string
	// TODO(ortutay): ID based policies
}

type Policy struct {
	Selector PolicySelector
	Cmd      PolicyCmd
	Args     []interface{}
}

type Conf struct {
	Policies []Policy
	BtcAddr  BtcAddr // TODO(ortutay): do not rely on address re-use
}

func (c *Conf) MatchingPolicies(service string, method string) []*Policy {
	fmt.Printf("compare %v %v to policies: %v\n", service, method, c.Policies)
	matching := make([]*Policy, 0)
	for i, policy := range c.Policies {
		if policy.Selector.Service != "" &&
			policy.Selector.Service != service {
			continue
		}
		if policy.Selector.Method != "" &&
			policy.Selector.Method != method {
			continue
		}
		matching = append(matching, &c.Policies[i])
	}
	fmt.Printf("matching: %v\n", matching)
	return matching
}
