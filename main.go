// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package main

import (
	"fmt"
	"os"

	"atomizer.io/cmd"
	_ "atomizer.io/montecarlopi"
)

func main() {
	err := cmd.Initialize("Atomizer Test Agent")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
