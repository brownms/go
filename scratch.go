package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

func main() {
	fmt.Printf("empty => [%v]", strings.Join([]string{}, ","))

	scannerBytes()
	intToHex()
	regExpExample()
	scanBytes()

}

func scannerBytes() {
	fmt.Println("scannerBytes ========>")
	reader := bytes.NewReader([]byte("foo bar baz"))
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanBytes)

	empty := []byte{}
	byte_count := 0
	var bytes_seen []byte
	for scanner.Scan() {
		next := scanner.Bytes()
		count := len(next)
		byte_count += count
		bytes_seen = bytes.Join([][]byte{bytes_seen, next}, empty)
		fmt.Printf("Count: %v; Total: %v; Bytes Seen => [%v]\n", count, byte_count, string(bytes_seen))
	}

	letters := []string{"I", "Q"}
	for _, v := range letters {
		fmt.Printf("String: [%v]; Hex: [%x]; Binary: [%b]\n", v, v, []byte(v))
	}
}
func intToHex() {
	v := []int32{
		40000, 40960, 41000, 42000, 5, 5000, 988, 1499, 1500, 2000000000, 50001, 50100, 50200,
	}
	fmt.Println("intToHex ========>")
	for _, value := range v {
		fmt.Printf("Value Int::Hex ==> %v::%x\n", value, value)
	}
}

type RegexpCheck struct {
	pattern string
	value   string
}

func regExpExample() {
	checks := []RegexpCheck{
		{pattern: `foo.*`, value: "seafood"},
		{pattern: `^[A-Za-z0-9]*$`, value: "foobarbazBUZZ762w762"},
		{pattern: `^[A-Za-z0-9]*$`, value: "foo_barbazBUZ+Z762w762"},
		{pattern: "(7[a-zA-Z0-9]{24,34})", value: "7F1u3wSD5RbOHQmupo9nx4TnhQ"},
		{pattern: "(7[a-zA-Z0-9]{24,34})", value: "7F1uFFFFFFFFFFFFF000000000"},
		{pattern: "(7[a-zA-Z0-9]{24,34})", value: "7iKDZEwPZSqIvDnHvVN2r0hUWXD5rHX"},
		{pattern: "(7[a-zA-Z0-9]{24,34})", value: "7LOrwbDlS8NujgjddyogWgIM93MV5N2VR"},
		{pattern: "(7[a-zA-Z0-9]{24,34})", value: "7adNeSwJkMakpEcln9HEtthSRtxdmEHOT8T"},
		{pattern: "^7([a-zA-Z0-9]{24,34})", value: "FI7wNlVcf5hrqJECoN4bq4uNZgk"},
	}

	for _, check := range checks {
		matched, err := regexp.Match(check.pattern, []byte(check.value))
		fmt.Printf("Regexp: %v Matched: %v, Value: [%v]; Error: %v\n", check.pattern, matched, check.value, err)
	}
}

func scanBytes() {
	reader := strings.NewReader("this is a very long string that has many characters and just as many bytes")
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanBytes)
	i := 0
	for scanner.Scan() {
		b := scanner.Bytes()
		fmt.Printf("byte[%v] => %v [%x]\n", i, string(b), b)
		i += 1
	}

}
