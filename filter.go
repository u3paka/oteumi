package main

import (
	"fmt"
	"regexp"
	"strings"
)

func Kusoripu(t string) bool {
	emojiReg := regexp.MustCompile("[\U0001F0CF-\U000207BF]")
	symbolReg := regexp.MustCompile("[!-/:-@[-`{-~]")
	fmt.Println(t)
	if len(t) < 100 {
		return false
	}
	if strings.Contains(t, "http") {
		return false
	}
	if strings.Contains(t, "#") {
		return false
	}
	if strings.Count(t, "\u3000") > 5 {
		return true

	}
	if strings.Count(t, "\t") < 5 {
		return false
	}

	emc := len(emojiReg.Copy().FindAllStringIndex(t, -1))
	fmt.Println(emc)
	if emc > 5 {
		return true
	}

	symc := len(symbolReg.Copy().FindAllStringIndex(t, -1))
	fmt.Println(symc)
	if symc < 5 {
		return false
	}
	if symc > 20 {
		return true
	}
	return false
}
