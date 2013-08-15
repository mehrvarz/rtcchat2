package main

import (
	"fmt"
	"github.com/mehrvarz/rtcchat2"
)

var TAG = "gkvDisplay"

func main() {
	fmt.Println(TAG, "start...")
	rtcchat2.GkvInit()

	var user rtcchat2.UserInfo
	var key string

	key="f3e2-01d2-1f66-0cd3"; 
	user = rtcchat2.GkvGet(key)
	fmt.Println(TAG,key,user.Value,user.Created,user.LastAccessed,user.Counter)

	key="01d2-f3e2-d332-164d"; 
	user = rtcchat2.GkvGet(key)
	fmt.Println(TAG,key,user.Value,user.Created,user.LastAccessed,user.Counter)

	key="e6a2-dde2-d1f2-1622"; 
	user = rtcchat2.GkvGet(key)
	fmt.Println(TAG,key,user.Value,user.Created,user.LastAccessed,user.Counter)

	key="d9a6-c4e4-2231-8a3d"; 
	user = rtcchat2.GkvGet(key)
	fmt.Println(TAG,key,user.Value,user.Created,user.LastAccessed,user.Counter)
}

