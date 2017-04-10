package parser

import (
	"bufio"
	"os"
)

func createScanner(file *os.File) *bufio.Scanner {
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024*20) // 20MB
	return scanner
}
