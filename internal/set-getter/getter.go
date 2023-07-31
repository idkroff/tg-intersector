package getter

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetSet() []string {
	set := []string{}

	fmt.Println("Enter chat ids for set: (\"stop\" to stop)")
	reader := bufio.NewReader(os.Stdin)
	for {
		inp, _ := reader.ReadString('\n')
		inp = strings.Trim(inp, "\n")
		if inp == "stop" {
			break
		}

		set = append(set, inp)
	}

	return set
}
