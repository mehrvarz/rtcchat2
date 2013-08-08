// rtcchat2 calleeService.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtcchat2

import (
	"fmt"
)

var TAG6 = "CalleePersistKey"

func getPersistedCallerKey(calleeKey string) string {
	// TODO: convert calleeKey to callerKey
	fmt.Println(TAG6, "convert calleeKey="+calleeKey+" to callerKey ...")
	callerKey := calleeKey // DUMMY
	return callerKey
}

