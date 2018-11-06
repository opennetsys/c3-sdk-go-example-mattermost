package sqlstore

import (
	"github.com/c3systems/mattermost-server/store/storetest"
	"testing"
)

func TestTermsOfServiceStore(t *testing.T) {
	StoreTest(t, storetest.TestTermsOfServiceStore)
}
