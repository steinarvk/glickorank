// lines contains utilities for dealing with small line-oriented text files.
package lines

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

func OnLines(r io.Reader, cb func(string) (bool, error)) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) == "" {
			continue
		}
		cont, err := cb(text)
		if err != nil {
			return err
		}
		if !cont {
			break
		}
	}
	return nil
}

func ReadN(r io.Reader, limit int) ([]string, error) {
	var rv []string
	if err := OnLines(r, func(line string) (bool, error) {
		rv = append(rv, line)
		return len(rv) != limit, nil
	}); err != nil {
		return nil, err
	}
	return rv, nil
}

func Read(r io.Reader) ([]string, error) {
	return ReadN(r, -1)
}

func ReadFile(filename string) ([]string, error) {
	return ReadFileN(filename, -1)
}

func ReadFileN(filename string, limit int) ([]string, error) {
	lines, err := readFile(filename, limit)
	if err != nil {
		return nil, fmt.Errorf("error reading lines from %q: %v", filename, err)
	}
	return lines, nil
}

func readFile(filename string, limit int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	return ReadN(file, limit)
}

func AsMap(lines []string) map[string]bool {
	m := map[string]bool{}
	for _, line := range lines {
		m[line] = true
	}
	return m
}

func AsBytes(lines []string) []byte {
	var buf bytes.Buffer
	for _, line := range lines {
		fmt.Fprintf(&buf, "%s\n", line)
	}
	return buf.Bytes()
}

func Sub(a, b []string) []string {
	skip := AsMap(b)

	var rv []string
	for _, x := range a {
		if !skip[x] {
			rv = append(rv, x)
		}
	}

	return rv
}

func appendToFile(filename string, lines []string) error {
	writeData := AsBytes(lines)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return fmt.Errorf("error opening %q for appending: %v", filename, err)
	}

	n, err := f.Write(writeData)
	if err != nil {
		return fmt.Errorf("error writing to %q: %v", filename, err)
	}
	if n < len(writeData) {
		return fmt.Errorf("error writing to %q: short write (wrote %d/%d)", filename, n, len(writeData))
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing %q after write: %v", filename, err)
	}

	return nil
}

func AppendToFile(filename string, lines []string) error {
	if err := appendToFile(filename, lines); err != nil {
		return fmt.Errorf("error appending lines to file %q: %v", filename, err)
	}
	return nil
}

func AddNewToFile(filename string, maybeNewLines []string) error {
	existingLines, err := ReadFile(filename)
	if err != nil {
		return err
	}

	newLines := Sub(maybeNewLines, existingLines)

	if len(newLines) == 0 {
		return nil
	}

	return AppendToFile(filename, newLines)
}

func overwriteFile(filename string, lines []string) error {
	data := AsBytes(lines)

	if err := ioutil.WriteFile(filename, data, 0666); err != nil {
		return fmt.Errorf("error overwriting %q: %v", filename, err)
	}

	return nil
}

func OverwriteFile(filename string, lines []string) error {
	if err := overwriteFile(filename, lines); err != nil {
		return fmt.Errorf("error overwriting file %q: %v", filename, err)
	}
	return nil
}

func CreateOrExpect(filename string, lines []string, allowEmpty bool) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return OverwriteFile(filename, lines)
	}

	existingLines, err := ReadFile(filename)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existingLines, lines) {
		if len(existingLines) == 0 && allowEmpty {
			return OverwriteFile(filename, lines)
		}
		return fmt.Errorf("expected %q to contain %q if it existed, but it contained %q", filename, lines, existingLines)
	}

	return nil
}

func RemoveFromFile(filename string, linesToRemove []string, deleteFileIfEmpty bool) error {
	if len(linesToRemove) == 0 {
		return nil
	}

	existingLines, err := ReadFile(filename)
	if err != nil {
		return err
	}

	remainingLines := Sub(existingLines, linesToRemove)

	if len(remainingLines) == 0 && deleteFileIfEmpty {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("error deleting %q: %v", filename, err)
		}
		return nil
	}

	return OverwriteFile(filename, remainingLines)
}
