package scenario

import "github.com/anil-debug/rackpilot-ai/simulator/bmc/domain"

func errInvalid() error {
	return domain.ErrInvalidID
}
