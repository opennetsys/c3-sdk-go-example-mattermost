package sqlstore

import (
	"testing"

	"github.com/c3systems/c3-sdk-go-example-mattermost/store/storetest"
)

func TestTermsOfServiceStore(t *testing.T) {
	StoreTest(t, storetest.TestTermsOfServiceStore)
}
