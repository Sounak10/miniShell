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
	var outputFile string
	var fileDescriptor int = 1
	var appendMode bool = false

	redirectPos := strings.Index(cmd, ">>")
	if redirectPos != -1 {
		appendMode = true
	} else {

		redirectPos = strings.Index(cmd, ">")
	}

	if redirectPos != -1 {

		fdEndPos := redirectPos

		if redirectPos > 0 && cmd[redirectPos-1] >= '0' && cmd[redirectPos-1] <= '9' {
			fdStr := ""
			i := redirectPos - 1

			for i >= 0 && cmd[i] >= '0' && cmd[i] <= '9' {
				fdStr = string(cmd[i]) + fdStr
				i--
			}

			if fd, err := strconv.Atoi(fdStr); err == nil {
				fileDescriptor = fd
				fdEndPos = i + 1
			}
		}

		redirectCmd := cmd[:fdEndPos]
		var filenamePart string
		if appendMode {

			filenamePart = strings.TrimSpace(cmd[redirectPos+2:])
		} else {
			filenamePart = strings.TrimSpace(cmd[redirectPos+1:])
		}

		filename := strings.Builder{}
		inQuotes := false
		var quoteChar byte = 0

		for i := range len(filenamePart) {
			char := filenamePart[i]
			if (char == '"' || char == '\'') && (i == 0 || filenamePart[i-1] != '\\') {
				if !inQuotes {
					inQuotes = true
					quoteChar = char
				} else if char == quoteChar {
					inQuotes = false
				} else {
					filename.WriteByte(char)
				}
			} else if !inQuotes && char == ' ' {
				if filename.Len() > 0 {
					break
				}
			} else {
				filename.WriteByte(char)
			}
		}

		outputFile = filename.String()
		cmd = redirectCmd
	}

	command, args := getCmdAndArgs(cmd)

	if outputFile != "" {
		executeWithRedirection(command, args, outputFile, fileDescriptor, appendMode)
		return
	}

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

func executeWithRedirection(command string, args []string, outputFile string, fileDescriptor int, appendMode bool) {

	var file *os.File
	var err error

	if appendMode {
		file, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(outputFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %s\n", err)
		return
	}
	defer file.Close()

	var origStdout *os.File
	var origStderr *os.File

	switch fileDescriptor {
	case 1:
		origStdout = os.Stdout
		os.Stdout = file
	case 2:
		origStderr = os.Stderr
		os.Stderr = file
	default:
		fmt.Fprintf(os.Stderr, "Unsupported file descriptor: %d\n", fileDescriptor)
		return
	}

	if comm, ok := builtins[command]; !ok {
		cmd := exec.Command(command, args...)

		if fileDescriptor == 1 {
			cmd.Stdout = file
			cmd.Stderr = os.Stderr
		} else if fileDescriptor == 2 {
			cmd.Stdout = os.Stdout
			cmd.Stderr = file
		}

		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {

			if exitErr, ok := err.(*exec.Error); ok && exitErr.Err == exec.ErrNotFound {

				fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
			}
		}
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

	if origStdout != nil {
		os.Stdout = origStdout
	}
	if origStderr != nil {
		os.Stderr = origStderr
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
		if exitErr, ok := err.(*exec.Error); ok && exitErr.Err == exec.ErrNotFound {
			fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd)
		}
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
