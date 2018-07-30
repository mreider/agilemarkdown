package actions

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirmAction(question string) bool {
	fmt.Println(question)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))
	return text == "y"
}

func existsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
