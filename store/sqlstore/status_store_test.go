// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package sqlstore

import (
	"testing"

	"github.com/c3systems/c3-sdk-go-example-mattermost/store/storetest"
)

func TestStatusStore(t *testing.T) {
	StoreTest(t, storetest.TestStatusStore)
}
