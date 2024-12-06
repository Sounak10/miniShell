package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var builtins = map[string]int{"exit": 0, "echo": 1, "type": 2, "pwd": 3, "cd": 4}

func main() {

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "$ ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSuffix(cmd, "\n")
		if len(cmd) == 0 {
			continue
		}
		handleCmd(cmd)
	}

}

func handleCmd(cmd string) {
	command, args := getCmdAndArgs(cmd)
	if comm, ok := builtins[command]; !ok {
		execHandler(command, args)
	} else {
		switch comm {
		case 0:
			exitHandler(args)
		case 1:
			echoHandler(args)
		case 2:
			typeHandler(args)
		case 3:
			pwdHandler()
		case 4:
			cdHandler(args)
		}
	}
}

func exitHandler(args []string) {
	if len(args) == 0 {
		os.Exit(0)
	}
	code, _ := strconv.Atoi(args[0])
	os.Exit(code)
}

func echoHandler(args []string) {
	fmt.Println(strings.Join(args, " "))
}

func typeHandler(args []string) {
	item := args[0]
	if _, ok := builtins[item]; ok {
		fmt.Printf("%s is a shell builtin\n", item)
	} else {
		env := os.Getenv("PATH")
		paths := strings.Split(env, ":")
		for _, path := range paths {
			execPath := path + "/" + item
			if _, err := os.Stat(execPath); err == nil {
				fmt.Printf("%s is %s\n", item, execPath)
				return
			}
		}
		fmt.Printf("%s: not found\n", item)
	}

}

func pwdHandler() {
	path, _ := os.Getwd()
	fmt.Println(path)
}

func cdHandler(args []string) {
	if len(args) == 0 {
		os.Chdir(os.Getenv("HOME"))
	} else {
		path := strings.Split(args[0], "/")
		if path[0] == "~" {
			path[0] = os.Getenv("HOME")
			args[0] = strings.Join(path, "/")
		}
		err := os.Chdir(args[0])
		if err != nil {
			fmt.Printf("cd: %s: No such file or directory\n", args[0])
		}
	}
}

func execHandler(cmd string, args []string) {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	err := command.Run()
	if err != nil {
		fmt.Printf("%s: command not found\n", cmd)
	}

}

func getCmdAndArgs(cmd string) (string, []string) {
	var args []string
	var currentArg strings.Builder
	inSingleQuotes := false
	inDoubleQuotes := false
	escapeNext := false

	for i := 0; i < len(cmd); i++ {
		char := rune(cmd[i])

		if escapeNext {
			if inDoubleQuotes && !isSpecialChar(char) {
				currentArg.WriteRune('\\')
			}
			currentArg.WriteRune(char)
			escapeNext = false
		} else if char == '\\' {
			if inSingleQuotes {
				currentArg.WriteRune(char)
			} else {
				escapeNext = true
			}
		} else if char == '\'' && !inDoubleQuotes {
			inSingleQuotes = !inSingleQuotes
		} else if char == '"' && !inSingleQuotes {
			inDoubleQuotes = !inDoubleQuotes
		} else if char == ' ' && !inSingleQuotes && !inDoubleQuotes {
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
		} else {
			currentArg.WriteRune(char)
		}

		if i == len(cmd)-1 && escapeNext {
			currentArg.WriteRune('\\')
		}
	}

	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args[0], args[1:]

}

func isSpecialChar(char rune) bool {
	return char == '\\' || char == '$' || char == '"' || char == '\n'
}
