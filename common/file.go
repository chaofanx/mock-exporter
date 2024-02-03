package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func ReadFileAsync(path *string) (<-chan string, error) {
	lines := make(chan string)

	go func() {
		defer close(lines)

		file, err := os.Open(*path)
		if err != nil {
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			lines <- line
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			lines <- fmt.Sprintf("Error reading file: %v", err)
		}
	}()

	return lines, nil
}

func ReadFile(path *string) string {
	file, err := os.ReadFile(*path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}
	return string(file)
}
