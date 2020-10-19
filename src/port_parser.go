/*package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Println("Error provide at least 2 arguments!")
		os.Exit(1)
	}

	num1, _ := strconv.Atoi(args[0])
	num2, _ := strconv.Atoi(args[1])

	fmt.Printf("Port: %d\n", getPort(num1, num2))

}

func getPort(value1, value2 int) uint16 {
	return ((uint16(value1) | 0) << 8) | uint16(value2)
}
*/
package main
