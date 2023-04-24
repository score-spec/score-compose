/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package main

import (
	"fmt"
	"os"

	"github.com/score-spec/score-compose/internal/command"
)

func main() {
	status := command.ParseOpts()
	if status == 0 {
		if err := command.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

}
