package uuid

import (
	"fmt"
	"regexp"
)

const (
	uuidPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)

var UUIDRegex = regexp.MustCompile(uuidPattern)

func Validate(uuid string) error {
	if !UUIDRegex.MatchString(uuid) {
		return fmt.Errorf("incorrect UUID")
	}
	return nil
}
