package sqlstore

import (
	"github.com/c3systems/c3-sdk-go-example-mattermost/store/storetest"
	"testing"
)

func TestTermsOfServiceStore(t *testing.T) {
	StoreTest(t, storetest.TestTermsOfServiceStore)
}
