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

func verifyDir(ctx *cli.Context) error {
	//  get all valid folders to create a go module
	wdp := "/Users/tashi/go/src/*/[0-9a-zA-Z]*" // pattern to match
	wdm, err := filepath.Glob(wdp)              // matches
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
