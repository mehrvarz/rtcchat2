// rtcchat2 calleePersistKey.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtcchat2

import (
	"bytes"
	"encoding/gob"
	"fmt"
    "github.com/steveyen/gkvlite"	// see: https://github.com/steveyen/gkvlite
	"math/rand"
    "os"
    "time"
)

var TAG6 = "CalleePersistKey"

type UserInfo struct {
	Value string
	Created time.Time
	LastAccessed time.Time
	Counter int
	AutoAnswer bool
	Flag2 bool
	Flag3 bool
	Int1 int
	Int2 int
	Int3 int
}

var F *os.File
var S *gkvlite.Store
var C *gkvlite.Collection
var enc *gob.Encoder
var dec *gob.Decoder
var	gkvCanWrite chan bool

func GkvInit() {
	fmt.Println(TAG6, "init open")
	var err error = nil
	F, err = os.OpenFile("rtcchat.gkv",os.O_RDWR,664)
	if(err!=nil) {
		fmt.Println(TAG6, "err",err)
		os.Exit(0)
	}

	fmt.Println(TAG6, "init gkvlite.NewStore(F)")
	S, err = gkvlite.NewStore(F)
	if(err!=nil) {
		fmt.Println(TAG6, "err",err)
		os.Exit(0)
	}

	fmt.Println(TAG6, "init s.SetCollection()")
	C = S.SetCollection("users", nil)

	// feed gkvCanWrite channel one element, to signal that access to c.Set() is possible
	//fmt.Println(TAG6, "init make(chan bool, 1)")
	gkvCanWrite = make(chan bool, 1)
	//fmt.Println(TAG6, "init make(chan bool, 1) done")
	gkvCanWrite <- true
	fmt.Println(TAG6, "init gkvCanWrite <- true done")
}

func GkvCreate() {
	fmt.Println(TAG, "GkvCreate")
	var err error = nil
	F, err = os.Create("rtcchat.gkv")
	if(err!=nil) {
		fmt.Println(TAG, "err",err)
		os.Exit(0)
	}
}

func BGCleaner() {
	var maxEntryIdleTime int64 = 60*60*24*31	// after 31 days of no use, delete callee key

	for {
		// every 30 minutes check for outdated entries
		select {
		case <-time.After(30*60 * time.Second):
			var deleteOnlyOne = false
			now := time.Now()
			fmt.Println(TAG6, "BGCleaner...")
			C.VisitItemsAscend([]byte(""), true, func(i *gkvlite.Item) bool {
				// This visitor callback will be invoked with every item
				// If we want to stop visiting, return false;
				key := string(i.Key)
				user := GkvGet(key)

				// show the create-age and idle-age in seconds
				ageCreated := now.Unix()-user.Created.Unix()
				ageIdle := now.Unix()-user.LastAccessed.Unix()
				if(ageIdle>maxEntryIdleTime && !deleteOnlyOne) {
					fmt.Println(TAG6, "BGCleaner Key",key,
										"Val",user.Value,
										"Crea",ageCreated,
										"Last",ageIdle,"C",user.Counter)
					//fmt.Println("Key",key,"is inactive too long; may be removed")
					<-gkvCanWrite
					C.Delete(i.Key)
					gkvCanWrite <- true

					fmt.Println(TAG6, "BGCleaner Key",key,"was inactive too long; has been removed")
					S.Flush()
					F.Sync()
					//deleteOnlyOne = true
				}
				return true
			})
		}
		fmt.Println(TAG6, "BGCleaner done")
	}
}

func GkvGet(key string) UserInfo {
	var user UserInfo
	var byteBuf bytes.Buffer
	//fmt.Println(TAG6, "GkvGet "+key)
	userArray, err := C.Get([]byte(key))
	if(err!=nil) {
		fmt.Println(TAG6, "GkvGet c.Get "+key+" err",err)
		return user
	}
	//fmt.Println("len(userArray)",len(userArray))

	// convert byte[] to byteBuf
	_, err = byteBuf.Write(userArray)
	if(err!=nil) {
		fmt.Println(TAG6, "GkvGet byteBuf.Write "+key+" err",err)
		return user
	}
	//fmt.Println("l",l)

	// decode byteBuf to UserInfo
	err = gob.NewDecoder(&byteBuf).Decode(&user)
	if(err!=nil) {
		fmt.Println(TAG6, "GkvGet dec.Decode "+key+" err",err)
		return user
	}
	//fmt.Println(TAG6,"GkvGet",key,user.Value,user.Created,user.LastAccessed,user.Counter)
	return user
}

func GkvSet(key string, user UserInfo) {
	// convert user into byteBuf, then write byteBuf using c.Set(key,array)
	fmt.Println(TAG6,"GkvSet",user.Value,user.Created,user.LastAccessed,user.Counter)
    var byteBuf bytes.Buffer
	gob.NewEncoder(&byteBuf).Encode(user)

	// note: multithread + persistence
	// this chan operation will block, if the gkvCanWrite channel is currently empty
	<-gkvCanWrite
	//fmt.Println(TAG6,"GkvSet c.Set()...")
	C.Set([]byte(key), byteBuf.Bytes())
	// feed gkvCanWrite channel one element, to signal that access to c.Set() is possible
	gkvCanWrite <- true

	S.Flush()
	F.Sync()
	fmt.Println(TAG6,"GkvSet done")
}

func getPersistedCallerKey(calleeKey string) string {
	user := GkvGet(calleeKey)
	if(user.Value!="") {
		user.LastAccessed = time.Now()
		user.Counter = user.Counter +1
		GkvSet(calleeKey,user)
	}
	return user.Value
}

func CreateNewKeys() (string,string) {
	for {
		calleeKey := generateCalleeKey()
		callerKey := generateCallerKey()
		user := GkvGet(calleeKey)
		if(user.Value=="") {
			// TODO: URGENT: also need to check that callerKey does nots exit
			return calleeKey, callerKey
		}
		fmt.Println(TAG6,"CreateNewKeys calleeKey="+calleeKey+" exists - try again")
	}
}

func StoreNewKeys(calleeKey string, callerKey string) {
	var nowTime = time.Now()
	GkvSet(calleeKey, UserInfo{callerKey, nowTime, nowTime, 1, false, false, false, 0, 0, 0})
}

func generateCalleeKey() string {
	var S4 = func() string {
		return fmt.Sprintf("%x", ((1 + rand.Intn(0x10000)) | 0))[1:]
	}
	return (S4() + S4() + "-" + S4() + "-" + S4() + "-" + S4() + "-" + S4() + S4() + S4())
}

func generateCallerKey() string {
	var S4 = func() string {
		return fmt.Sprintf("%x", ((1 + rand.Intn(0x10000)) | 0))[1:]
	}
	return (S4() + S4() + "-" + S4() + "-" + S4() + "-" + S4() + "-" + S4() + S4() + S4())
}

