package main

import (
	jsonata "github.com/blues/jsonata-go"
	"github.com/kr/pretty"
	"github.com/pebbe/util"

	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TestT struct {
	Bindings        interface{} `json:"bindings"`
	Category        string      `json:"category"`
	Code            string      `json:"code"`
	Data            interface{} `json:"data"`
	Dataset         string      `json:"dataset"`
	Depth           int         `json:"depth"`
	Description     string      `json:"description"`
	Error           interface{} `json:"error"`
	Expr            string      `json:"expr"`
	ExprFile        string      `json:"expr-file"`
	Function        string      `json:"function"`
	Result          interface{} `json:"result"`
	Timelimit       int         `json:"timelimit"`
	Token           string      `json:"token"`
	UndefinedResult bool        `json:"undefinedResult"`
	Unordered       bool        `json:"unordered"`
}

var (
	x = util.CheckErr
	w = util.WarnErr

	tests = make([]string, 0)

	//base = "my-test-suite"
	base = "jsonata/test/test-suite"
)

func main() {
	scan(filepath.Join(base, "groups"))

	for _, test := range tests {
		doTest(test)
	}
}

func scan(dirname string) {
	entries, err := os.ReadDir(dirname)
	x(err)
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		fullname := filepath.Join(dirname, name)
		if entry.IsDir() {
			scan(fullname)
		} else if entry.Type().IsRegular() && strings.HasSuffix(name, ".json") {
			tests = append(tests, fullname)
		}
	}
}

func doTest(filename string) {
	var tests []*TestT
	data, err := os.ReadFile(filename)
	x(err)
	err = json.Unmarshal(data, &tests)
	if err == nil {
		for _, test := range tests {
			doOneTest(filename, test)
		}
		return
	}
	var test TestT
	x(json.Unmarshal(data, &test))
	doOneTest(filename, &test)

}

func doOneTest(filename string, test *TestT) {
	fmt.Println("\t---", strings.Replace(filename, base+"/", "...", 1))

	if test.ExprFile != "" {
		b, err := os.ReadFile(filepath.Join(filepath.Dir(filename), test.ExprFile))
		x(err)
		test.Expr = string(b)
	}

	if test.Dataset != "" {
		b, err := os.ReadFile(filepath.Join(base, "datasets", test.Dataset+".json"))
		x(err)
		x(json.Unmarshal(b, &(test.Data)))
	}

	if test.Code != "" && test.Depth > 0 {
		w(fmt.Errorf("Skipping test"))
		pretty.Println(test)
		return
	}

	e, err := jsonata.Compile(test.Expr)
	if err != nil {
		if test.Code == "" {
			w(err, "(compile)")
			pretty.Println(test)
		}
		return
	}

	res, err := e.Eval(test.Data)

	if test.Code != "" {
		if err == nil {
			w(fmt.Errorf("Expected error %s", test.Code), "(eval)")
			pretty.Println(test)
		}
		return
	}

	if test.UndefinedResult {
		if err == nil /* || err.Error() != "no results found" */ {
			w(fmt.Errorf("No results expected, got %#v", res), "(eval)")
			pretty.Println(test)
		}
		return
	}

	if test.Error != nil {
		if err == nil {
			w(fmt.Errorf("Error expected, got %#v", res), "(eval)")
			pretty.Println(test)
		}
		return
	}

	if w(err, "(eval)") != nil {
		pretty.Println(test)
		return
	}

	b, err := json.MarshalIndent(test.Result, "", "  ")
	x(err)
	expected := string(b)
	b, err = json.MarshalIndent(res, "", "  ")
	x(err)
	got := string(b)
	if got != expected {
		if test.Unordered == false || !orderedEqual(got, expected) {
			fmt.Println("EXPECTED:")
			fmt.Println(expected)
			fmt.Println("GOT:")
			fmt.Println(got)
			pretty.Println(test)
		}
	}
}

func orderedEqual(s1, s2 string) bool {
	ss1 := strings.Split(s1, "\n")
	ss2 := strings.Split(s2, "\n")
	for i, s := range ss1 {
		ss1[i] = strings.TrimRight(s, ",\n")
	}
	for i, s := range ss2 {
		ss2[i] = strings.TrimRight(s, ",\n")
	}
	sort.Strings(ss1)
	sort.Strings(ss2)
	s1 = strings.Join(ss1, "\n")
	s2 = strings.Join(ss2, "\n")
	return s1 == s2
}
