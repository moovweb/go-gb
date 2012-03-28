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
	//"errors"
	"fmt"
	"os"
	"exec"
	"path/filepath"
	"strings"
	"bytes"
)

var MakeCMD,
	CompileCMD,
	CCMD,
	AsmCMD,
	LinkCMD,
	PackCMD,
	CopyCMD,
	GoInstallCMD,
	GoFMTCMD,
	GoFixCMD,
	CGoCMD,
	GCCCMD,
	ProtocCMD,
	PkgconfigCMD string

func FindGobinExternal(name string) (path string, err error) {
	path, err = exec.LookPath(name)
	if err != nil {
		path = filepath.Join(GOBIN, name)
		_, err = os.Stat(path)
	}
	return
}

func FindExternals() (err error) {

	CompileCMD, err = FindGobinExternal(GetCompilerName())
	if err != nil {
		fmt.Printf("Could not find '%s' in path\n", GetCompilerName())
		return
	}
	AsmCMD, err = FindGobinExternal(GetAssemblerName())
	if err != nil {
		fmt.Printf("Could not find '%s' in path\n", GetAssemblerName())
		return
	}
	LinkCMD, err = FindGobinExternal(GetLinkerName())
	if err != nil {
		fmt.Printf("Could not find '%s' in path\n", GetLinkerName())
		return
	}
	PackCMD, err = FindGobinExternal("gopack")
	if err != nil {
		fmt.Printf("Could not find 'gopack' in path\n")
		return
	}

	var err2 error
	CGoCMD, err2 = FindGobinExternal("cgo")
	if err2 != nil {
		fmt.Printf("Could not find 'cgo' in path\n")
	}
	MakeCMD, err2 = FindGobinExternal("gomake")
	if err2 != nil {
		fmt.Printf("Could not find 'gomake' in path\n")
	}
	GoInstallCMD, err2 = FindGobinExternal("goinstall")
	if err2 != nil {
		fmt.Printf("Could not find 'goinstall' in path\n")
	}
	GoFMTCMD, err2 = FindGobinExternal("gofmt")
	if err2 != nil {
		fmt.Printf("Could not find 'gofmt' in path\n")
	}
	GoFixCMD, err2 = FindGobinExternal("gofix")
	if err2 != nil {
		fmt.Printf("Could not find 'gofix' in path\n")
	}
	GCCCMD, err2 = exec.LookPath("gcc")
	if err2 != nil {
		//fmt.Printf("Could not find 'gcc' in path\n")
	}
	CCMD, err2 = FindGobinExternal(GetCCompilerName())
	if err2 != nil {
		fmt.Printf("Could not find '%' in path\n", GetCCompilerName())
	}
	ProtocCMD, err2 = exec.LookPath("protoc")
	if err2 != nil {
		//fmt.Printf("Could not find 'protoc' in path\n")
	}
	PkgconfigCMD, err2 = exec.LookPath("pkg-config")
	if err2 != nil {
		fmt.Printf("Could not find 'pkg-config' in path\n")
	}

	CopyCMD, _ = exec.LookPath("cp")

	return
}

func SplitArgs(args []string) (sargs []string) {
	for _, arg := range args {
		sarg := strings.Fields(arg)
		sargs = append(sargs, sarg...)
	}
	return
}

func RunExternalDump(cmd, wd string, argv []string, dump *os.File) (err error) {
	argv = SplitArgs(argv)

	if Verbose {
		fmt.Printf("%s\n", argv)
	}

	c := exec.Command(cmd, argv[1:]...)
	c.Dir = wd
	c.Env = os.Environ()

	c.Stdout = dump
	c.Stderr = os.Stderr

	err = c.Run()

	if wmsg, ok := err.(*os.Waitmsg); ok {
		if wmsg.ExitStatus() != 0 {
			err = errors.New(fmt.Sprintf("%v: %s\n", argv, wmsg.String()))
		} else {
			err = nil
		}
	}
	return
}
func RunExternal(cmd, wd string, argv []string) (err error) {
	return RunExternalDump(cmd, wd, argv, os.Stdout)
}
func RunExternalAndStdout(cmd, wd string, argv []string) (stdout []byte, err os.Error) {
	argv = SplitArgs(argv)
	var c *exec.Cmd
	if len(argv) > 0 {
		c = exec.Command(cmd, argv...)
	} else {
		c = exec.Command(cmd)
	}
	c.Dir = wd
	c.Env = os.Environ()

	bStdout := bytes.NewBuffer(nil)
	bStderr := bytes.NewBuffer(nil)
	c.Stdout = bStdout
	c.Stderr = bStderr

	err = c.Run()

	if wmsg, ok := err.(*os.Waitmsg); ok {
		if wmsg.ExitStatus() != 0 {
			err = os.NewError(fmt.Sprintf("command: %s %v\n%s\n", cmd, argv, bStderr.String()))
		} else {
			err = nil
		}
	} else {
		stdout = bStdout.Bytes()
	}
	return
}
