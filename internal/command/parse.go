package command

import (
	"fmt"
	"os"
	"strings"
)

type Usage struct {
	Flags []string
}

var commandMap map[string]*Usage
var invalidFlags []string

func ParseOpts() int {

	commandMap = map[string]*Usage{
		"score-compose": {
			Flags: []string{"h", "v", "help", "version"},
		},
		"run": {

			Flags: []string{"h", "o", "f", "build", "env-file", "file", "help", "output", "overrides", "verbose"},
		},
		"help": {
			Flags: []string{},
		},
		"version": {
			Flags: []string{},
		},
	}

	first_arg := os.Args[1]
	checkIfValidScoreComposeFlags(first_arg, commandMap["score-compose"].Flags)
	ok2 := checkIfValidScoreComposeCommand(first_arg, commandMap)
	if ok2 {
		checkForValidFlags(os.Args[1:], commandMap)
	}

	if len(invalidFlags) > 0 {

		fmt.Printf("Error: Unknown command/flags: ")
		for _, v := range invalidFlags {
			value := string(v)
			fmt.Printf("%v, ", value)
		}
		fmt.Printf("\nUse \"score-compose --help\" for more information.\n")
		return -1
	}
	return 0
}

func checkForValidFlags(args []string, commandMap map[string]*Usage) {
	validFlags := commandMap[args[0]].Flags
	for _, v := range args {
		value := string(v)
		if strings.HasPrefix(value, "--") {
			trim_arg := strings.TrimPrefix(value, "--")
			if !contains(trim_arg, validFlags) {
				if !contains(trim_arg, invalidFlags) {
					invalidFlags = append(invalidFlags, trim_arg)

				}

			}
		} else if strings.HasPrefix(value, "-") {
			trim_arg := strings.TrimPrefix(value, "-")
			for _, v := range trim_arg {
				flag := string(v)
				if !contains(flag, validFlags) {

					if !contains(flag, invalidFlags) {
						invalidFlags = append(invalidFlags, flag)
					}

				}
			}

		}

	}

}

func checkIfValidScoreComposeFlags(arg string, Flags []string) {
	if strings.HasPrefix(arg, "--") {
		trim_arg := strings.TrimPrefix(arg, "--")
		if !contains(trim_arg, Flags) {
			if !contains(trim_arg, invalidFlags) {
				invalidFlags = append(invalidFlags, trim_arg)
			}
		}
	} else if strings.HasPrefix(arg, "-") {
		trim_arg := strings.TrimPrefix(arg, "-")
		for _, v := range trim_arg {
			value := string(v)
			if !contains(value, Flags) {
				if !contains(value, invalidFlags) {
					invalidFlags = append(invalidFlags, value)
				}
			}
		}
	}
}

func contains(v string, elements []string) bool {
	for _, s := range elements {
		if v == s {
			return true
		}
	}
	return false
}

func checkIfValidScoreComposeCommand(arg string, commandMap map[string]*Usage) bool {
	if _, ok := commandMap[arg]; ok {
		return true
	}
	return false
}
