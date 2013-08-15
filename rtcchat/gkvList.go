// see: https://github.com/steveyen/gkvlite
package main

import (
	"fmt"
	"github.com/mehrvarz/rtcchat2"
    "github.com/steveyen/gkvlite"
    "time"
)

var TAG = "gkvList"

func main() {
	//var maxIdleTime int64 = 60*60*24	// 24 hours
	fmt.Println(TAG, "start...")
	rtcchat2.GkvInit()

	now := time.Now()
	rtcchat2.C.VisitItemsAscend([]byte(""), true, func(i *gkvlite.Item) bool {
		// This visitor callback will be invoked with every item
		// If we want to stop visiting, return false;
		key := string(i.Key)
		user := rtcchat2.GkvGet(key)

		// show the create-age and idle-age in seconds
		ageCreated := now.Unix()-user.Created.Unix()
		ageIdle := now.Unix()-user.LastAccessed.Unix()
		fmt.Println("Key",key,
					"Val",user.Value,
					"Crea",ageCreated,
					"Last",ageIdle,
					"C",user.Counter,
					user.AutoAnswer,user.Flag2,user.Flag3,user.Int1,user.Int2,user.Int3)
		/*if(ageIdle>maxIdleTime) {
			fmt.Println("Key",key,"is inactive too long; may be removed")
			rtcchat2.C.Delete(i.Key)
			rtcchat2.S.Flush()
			rtcchat2.F.Sync()
		}*/
		return true
	})
}

