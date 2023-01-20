package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

func main() {
	scannerBytes()
	intToHex()
	regExpExample()

	fmt.Printf("empty => [%v]", strings.Join([]string{}, ","))
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
	}

	for _, check := range checks {
		matched, err := regexp.Match(check.pattern, []byte(check.value))
		fmt.Printf("Regexp: %v Matched: %v, Error: %v\n", check.pattern, matched, err)
	}
}
