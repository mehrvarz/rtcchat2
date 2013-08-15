package main

import (
	"fmt"
	"github.com/mehrvarz/rtcchat2"
)

var TAG = "gkvCreate"

func main() {
	fmt.Println(TAG, "gkvCreate")
	// destructive! be careful running this
	rtcchat2.GkvCreate()
}

