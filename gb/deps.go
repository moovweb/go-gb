/*
   Copyright 2011 John Asmuth

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"bufio"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func GetDeps(source string) (pkg, target string, deps, funcs, cflags, ldflags []string, err error) {
	isTest := strings.HasSuffix(source, "_test.go") && Test
	var file *ast.File
	flag := parser.ParseComments
	if !isTest {
		flag = flag | parser.ImportsOnly
	}
	file, err = parser.ParseFile(token.NewFileSet(), source, nil, flag)
	if err != nil {
		return
	}

	w := &Walker{
		Name:       "",
		Target:     "",
		pkgPos:     0,
		Deps:       []string{},
		Funcs:      []string{},
		CGoLDFlags: []string{},
		CGoCFlags:  []string{},
		ScanFuncs:  isTest,
	}

	ast.Walk(w, file)

	deps = w.Deps
	pkg = w.Name
	target = w.Target
	funcs = w.Funcs
	cflags = RemoveDups(w.CGoCFlags)
	ldflags = RemoveDups(w.CGoLDFlags)

	return
}

func RemoveDups(list []string) (newlist []string) {
	m := make(map[string]bool)
	for _, item := range list {
		m[item] = true
	}
	newlist = make([]string, 0)
	for item, _ := range m {
		newlist = append(newlist, item)
	}
	return
}

type Walker struct {
	Name       string
	Target     string
	pkgPos     token.Pos
	Deps       []string
	Funcs      []string
	CGoLDFlags []string
	CGoCFlags  []string
	ScanFuncs  bool
}

func (w *Walker) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.File:
		w.Name = n.Name.Name
		w.pkgPos = n.Package
		return w
	case *ast.ImportSpec:
		w.Deps = append(w.Deps, string(n.Path.Value))
		return nil
	case *ast.Comment:
		if n.Pos() < w.pkgPos {
			text := string(n.Text)
			if !strings.HasPrefix(text, "//") {
				return nil
			}
			text = strings.TrimSpace(text[2:])

			if strings.HasPrefix(text, "target:") {
				fields := strings.Fields(text[len("target:"):])
				if len(fields) > 0 {
					w.Target = fields[0]
				}
				//w.Target = strings.TrimSpace(text[len("target:"):])
			}
		} else {

			handleCommentLine := func(text string) {
				if strings.HasPrefix(text, "#cgo") {
					cgoMsg := strings.TrimSpace(text[len("#cgo"):])

					fields := strings.Fields(cgoMsg)
					if len(fields) >= 1 {
						flag := fields[0]
						if !strings.HasSuffix(flag, ":") {
							if !CheckCGOFlag(flag) {
								return
							} else {
								cgoMsg = strings.TrimSpace(cgoMsg[len(flag):])
							}
						}
					}

					cflags := false
					lflags := false
					if strings.HasPrefix(cgoMsg, "CFLAGS:") {
						cflags = true
						cgoMsg = strings.TrimSpace(cgoMsg[len("CFLAGS:"):])
					} else if strings.HasPrefix(cgoMsg, "LDFLAGS:") {
						lflags = true
						cgoMsg = strings.TrimSpace(cgoMsg[len("LDFLAGS:"):])
					} else if strings.HasPrefix(cgoMsg, "pkg-config:") {
						cgoMsg = strings.TrimSpace(cgoMsg[len("pkg-config:"):])
						if len(fields) > 1 {
							args := make([]string, 2)
							args[0] = "--cflags"
							for _, field := range(fields[1:]) {
								args[1] = field
								stdout, err := RunExternalAndStdout("pkg-config", "", args)
								if err != nil {
									ErrLog.Printf("pkg-config err: %q\n", err.String())
								} else {
									w.CGoCFlags = append(w.CGoCFlags, strings.TrimSpace(string(stdout)))
								}
							}
							args[0] = "--libs"
							for _, field := range(fields[1:]) {
								args[1] = field
								stdout, err := RunExternalAndStdout("pkg-config", "", args)
								if err != nil {
									ErrLog.Printf("pkg-config err: %q\n", err.String())
								} else {
									w.CGoLDFlags = append(w.CGoLDFlags, strings.TrimSpace(string(stdout)))
								}
							}
						}
					}

					if cflags {
						w.CGoCFlags = append(w.CGoCFlags, cgoMsg)
					}
					if lflags {
						w.CGoLDFlags = append(w.CGoLDFlags, cgoMsg)
					}
				}
			}

			text := string(n.Text)
			if strings.HasPrefix(text, "//") {
				text = strings.TrimSpace(text[2:])
				handleCommentLine(text)
			} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
				text = strings.TrimSpace(text[2 : len(text)-2])
				br := bufio.NewReader(bytes.NewBuffer([]byte(text)))
				for {
					line, isprefix, err := br.ReadLine()
					if err != nil {
						break
					}
					if isprefix {
						ErrLog.Printf("Extra long comment ignored: %v", node.Pos())
						return
					}
					lines := string(line)
					lines = strings.TrimSpace(lines)
					handleCommentLine(lines)
				}
			}
		}
		return nil
	case *ast.FuncDecl:
		if w.ScanFuncs {
			fdecl, ok := node.(*ast.FuncDecl)
			if ok && fdecl.Recv == nil {
				w.Funcs = append(w.Funcs, fdecl.Name.Name)
			}
		}
		return nil
	case *ast.GenDecl, *ast.CommentGroup:
		return w
	}
	return nil
}
