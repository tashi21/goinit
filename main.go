package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

// print stdout and stderr
func printStd(o, e *bytes.Buffer) {
	if e.Len() > 0 {
		fmt.Println(e.String())
	}
	if o.Len() > 0 {
		fmt.Println(o.String())
	}

	// reset buffers
	o.Reset()
	e.Reset()
}

// create the go module
func createModule(ctx *cli.Context, dir string) error {
	var name string       // module name
	var o, e bytes.Buffer // stdout, stderr buffers

	// get name of the module, else default to "project"
	if ctx.NArg() > 0 {
		name = ctx.Args().Get(0)
	} else {
		name = "project"
	}

	// create directory for go module
	mkdir := exec.Command("mkdir", name)
	mkdir.Dir = dir
	mkdir.Stdout = &o
	mkdir.Stderr = &e
	err := mkdir.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	dir = filepath.Join(dir, name) // update dir after creating new folder

	// initialize go.mod
	init := exec.Command("go", "mod", "init")
	init.Dir = dir
	init.Stdout = &o
	init.Stderr = &e
	err = init.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	// initialize main.go
	data := []byte("package main\n\nfunc main() {}\n")
	err = os.WriteFile(dir+"/main.go", data, 0644)
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	// initialize git repository
	git := exec.Command("git", "init")
	git.Dir = dir
	git.Stdout = &o
	git.Stderr = &e
	err = git.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	//  create .gitignore file from .gitignore template in home
	gitignore := exec.Command("cp", os.Getenv("HOME")+"/.gitignore", ".gitignore")
	gitignore.Dir = dir
	gitignore.Stdout = &o
	gitignore.Stderr = &e
	err = gitignore.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	// add all files to git
	add := exec.Command("git", "add", ".")
	add.Dir = dir
	add.Stdout = &o
	add.Stderr = &e
	err = add.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	// commit all files
	commit := exec.Command("git", "commit", "-m", "initial commit")
	commit.Dir = dir
	commit.Stdout = &o
	commit.Stderr = &e
	err = commit.Run()
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return err
	}
	printStd(&o, &e)

	// open vscode
	vsc := exec.Command("code", ".")
	vsc.Dir = dir
	vsc.Stdout = &o
	vsc.Stderr = &e
	err = vsc.Run()
	if err != nil {
		log.Printf("%v: %v", err, e)
		return err
	}
	printStd(&o, &e)

	return nil
}

// display choices for valid directories
func choices(ctx *cli.Context, opts []string) error {
	fmt.Println("Choose destination:")
	for i, opt := range opts {
		fmt.Printf("[%d] %s\n", i, opt)
	}
	var opt int
	fmt.Print("> ")
	fmt.Scanln(&opt)
	if opt < 0 || opt > (len(opts)-1) {
		fmt.Println("Invalid choice.")
		return nil
	}
	dir := opts[opt]
	return createModule(ctx, dir)
}

// get the go path by executing go env GOPATH
func getGoPath() (string, error) {
	var o, e bytes.Buffer // stdout, stderr buffers

	// get go path
	gopath := exec.Command("go", "env", "GOPATH")
	gopath.Stdout = &o
	gopath.Stderr = &e
	//  reset buffers after running command
	defer func() {
		o.Reset()
		e.Reset()
	}()

	err := gopath.Run() // run command
	if err != nil {
		log.Printf("%v: %v", err, e.String())
		return "", err
	}

	gpth := o.String() // get go path
	// remove newline from go path
	gpth = gpth[:len(gpth)-1]

	if gpth == "" { // if go path is empty
		gpth = "~/go" // default to $HOME/go
	}

	return gpth, nil
}

// verify if current directory is a valid directory to create a go module in
func verifyDir(ctx *cli.Context) error {
	gopath, err := getGoPath() // get go path

	if err != nil {
		return err
	}
	//  get all valid directories to create a go module in
	wdp := fmt.Sprintf("%s/src/*/[0-9a-zA-Z]*", gopath) // pattern to match
	wdm, err := filepath.Glob(wdp)                      // matches
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	pwd, err := os.Getwd() // get pwd
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	pwd, err = filepath.EvalSymlinks(pwd) // resolve symlinks
	if err != nil {
		log.Printf("%v", err)
		return err
	}

	// check if pwd is a valid folder to create a go module in
	if slices.Contains(wdm, pwd) {
		return createModule(ctx, "")
	}

	return choices(ctx, wdm)
}

func main() {
	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "goinit",
		Usage:                "create go module",
		Action: func(ctx *cli.Context) error {
			return verifyDir(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
