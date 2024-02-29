/******** Peter Winzell (c), 2/28/24 *********************************************/
package utils

import (
	"testing"
)

// Testing UDS json file parsing
func TestUdsRegistration(t *testing.T) {
	udsRegList := ReadUdsRegistrations("../vissv2server/uds-registration.docker.json")
	for i, item := range udsRegList {
		t.Logf("Item %d: %s", i, item)
	}
}
