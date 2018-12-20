package application

import (
	"errors"
	"fmt"
	"github.com/Equanox/gotron/cmd/gotron-builder/internal/file"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/Benchkram/errz"
	"github.com/otiai10/copy"

	"github.com/Equanox/gotron"
)

// Globals constants
const (
	gotronBuilderDirectory = ".gotron-builder"
)

type App struct {
	GoEntryPoint string // Directory where go build is executed
	AppDir       string // Application loaded by electronjs
	Target       string // Target system to build for
	OutputDir    string // Outputdirectory for build output
	Arch		 string // Architecture to build for
}

type GoBuildOptions struct {
	GoEnv        map[string]string
	buildOptions map[string]string
}

func (app *App) Run() (err error) {
	defer errz.Recover(&err)

	// Use gotron-browser-window to copy webapp
	// to .gotron dir. Let it handle the necessary logic
	// to validate webapp.
	gbw, err := gotron.New(app.AppDir)
	err = gbw.CreateAppStructure()
	errz.Fatal(err)

	err = app.makeTempDir()
	errz.Fatal(err)

	err = app.installDependencies()
	errz.Fatal(err)

	err = app.buildElectron()
	errz.Fatal(err)

	err = app.syncDistDirs()
	errz.Fatal(err)

	err = app.buildGoCode()
	errz.Fatal(err)

	return err
}

func New() *App {
	app := App{}
	err := app.SetTarget(runtime.GOOS)
	errz.Log(err)

	return &app
}

//SetTarget sets the operation system to build the executable for
func (app *App) SetTarget(target string) (err error) {
	switch target {
	case "win":
		fallthrough
	case "windows":
		fallthrough
	case "win32":
		app.Target = "win"
	case "linux":
		app.Target = "linux"
	case "darwin":
		fallthrough
	case "mac":
		app.Target = "mac"
	default:
		return errors.New("Unkown build target " + target)
	}
	return
}

func (app *App) makeTempDir() (err error) {
	os.RemoveAll(gotronBuilderDirectory)
	return os.Mkdir(gotronBuilderDirectory, os.ModePerm)
}

func runCmd(runDir, command string, args ...string) (err error) {
	defer errz.Recover(&err)

	// fmt.Println(runDir)GoEntryPoint
	// fmt.Println(command)
	// fmt.Println(args)

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = runDir
	err = cmd.Start()

	errz.Fatal(err)

	err = cmd.Wait()
	errz.Fatal(err)

	return
}

func runCmdEnv(runDir, command string, envVars []string, args ...string) (err error) {
	defer errz.Recover(&err)

	// fmt.Println(runDir)GoEntryPoint
	// fmt.Println(command)
	// fmt.Println(args)

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = runDir
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, envVars...)
	err = cmd.Start()

	errz.Fatal(err)

	err = cmd.Wait()
	errz.Fatal(err)

	return
}

func (app *App) installDependencies() (err error) {

	args := []string{"install", "electron-builder", "--save-dev"}

	return runCmd(gotronBuilderDirectory, "npm", args...)
}

// buildElectron
func (app *App) buildElectron() (err error) {
	if !file.Exists(app.AppDir) {
		return errors.New(
			fmt.Sprintf(
				"Given application directory [%s] does not exist",
				app.AppDir,
			))
	}
	// contains

	projDir, err := filepath.Abs(filepath.Join(app.GoEntryPoint, ".gotron/"))

	var target string
	switch app.Target {
	case "win":
		target = "-w"
	case "linux":
		target = "-l"
	case "mac":
		target = "-m"
	default:
	}

	args := []string{target, "--" + app.Arch, "--dir", "--projectDir=" + projDir}

	runDir := gotronBuilderDirectory
	command := filepath.Join("node_modules/.bin/", "electron-builder")

	return runCmd(runDir, command, args...)
}

func (app *App) buildGoCode() (err error) {
	defer errz.Recover(&err)
	args := []string{"build", "-tags", "gotronpack"}
	runDir := app.GoEntryPoint
	command := "go"
	
	var env []string
	switch app.Target {
	case "win":
		env = append(env, "GOOS=windows")
	case "linux":
		env = append(env, "GOOS=linux")
	case "mac":
		env = append(env, "GOOS=darwin")
	default:
	}

	switch app.Arch {
	case "x64":
		env = append(env, "GOARCH=amd64")
	case "ia32":
		env = append(env, "GOARCH=386")
	case "armv7l":
		env = append(env, "GOARCH=arm")
		env = append(env, "GOARM=7")
	case "arm64":
		env = append(env, "GOARCH=arm")
	default:
	}

	fName := filepath.Base(runDir)

	if app.Target == "win" {
		fName = fName + ".exe"
	}

	err = runCmdEnv(runDir, command, env, args...)
	errz.Fatal(err)

	from := filepath.Join(runDir, fName)
	var distFolder string
	if app.Arch == "x64" {
		distFolder = app.Target+"-unpacked"
	} else {
		distFolder = app.Target+"-"+app.Arch+"-unpacked"
	}
	to := filepath.Join(app.OutputDir, "dist", distFolder, fName)
	return os.Rename(from, to)
}

// Will copy everythin from .gotron/dist to .dist
func (app *App) syncDistDirs() (err error) {
	defer errz.Recover(&err)

	var distFolder string
	if app.Arch == "x64" {
		distFolder = app.Target+"-unpacked"
	} else {
		distFolder = app.Target+"-"+app.Arch+"-unpacked"
	}

	src := filepath.Join(app.GoEntryPoint, ".gotron/dist", distFolder)
	dst := filepath.Join(app.OutputDir, "dist", distFolder, "electronjs")

	err = copy.Copy(src, dst)
	errz.Fatal(err)

	err = os.RemoveAll(filepath.Dir(src))
	errz.Fatal(err)

	return nil
}