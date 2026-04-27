package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/scaffold"
)

func createSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		yes          bool
		list         bool
		templateName string
		projectName  string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("create", stdout, stderr)
		case "--yes", "-y":
			yes = true
		case "--list":
			list = true
		default:
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy create: unknown flag %q", a)
			}
			if templateName == "" {
				templateName = a
			} else if projectName == "" {
				projectName = a
			} else {
				return 1, fmt.Errorf("bunpy create: too many arguments")
			}
		}
	}

	if list {
		for _, t := range scaffold.List() {
			fmt.Fprintf(stdout, "%-12s %s\n", t.Name, t.Description)
		}
		return 0, nil
	}

	if templateName == "" {
		return 1, fmt.Errorf("usage: bunpy create <template> <name> [--yes]")
	}
	if _, ok := scaffold.Lookup(templateName); !ok {
		return 1, fmt.Errorf("bunpy create: unknown template %q (run `bunpy create --list`)", templateName)
	}
	if projectName == "" {
		return 1, fmt.Errorf("usage: bunpy create <template> <name> [--yes]")
	}

	vars := scaffold.Vars{
		Name:        projectName,
		SnakeName:   scaffold.SnakeName(projectName),
		Description: "A bunpy project",
		Author:      gitAuthor(),
		PythonMin:   ">=3.11",
	}

	if !yes {
		vars = promptVars(stdout, os.Stdin, vars)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 1, fmt.Errorf("bunpy create: %w", err)
	}
	destDir := filepath.Join(cwd, projectName)

	created, err := scaffold.Render(templateName, vars, destDir)
	if err != nil {
		return 1, fmt.Errorf("bunpy create: %w", err)
	}

	fmt.Fprintf(stdout, "Created %s (%s)\n", projectName, templateName)
	for _, f := range created {
		fmt.Fprintf(stdout, "  %s/%s\n", projectName, f)
	}
	if templateName != "script" {
		fmt.Fprintf(stdout, "\nNext: cd %s && bunpy install\n", projectName)
	}
	return 0, nil
}

func promptVars(stdout io.Writer, in io.Reader, defaults scaffold.Vars) scaffold.Vars {
	scanner := bufio.NewScanner(in)
	prompt := func(label, def string) string {
		fmt.Fprintf(stdout, "%s [%s]: ", label, def)
		if scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); line != "" {
				return line
			}
		}
		return def
	}
	vars := defaults
	vars.Name = prompt("Project name", defaults.Name)
	vars.SnakeName = scaffold.SnakeName(vars.Name)
	vars.Description = prompt("Description", defaults.Description)
	vars.Author = prompt("Author", defaults.Author)
	vars.PythonMin = prompt("Python version constraint", defaults.PythonMin)
	return vars
}

func gitAuthor() string {
	name, _ := exec.Command("git", "config", "user.name").Output()
	email, _ := exec.Command("git", "config", "user.email").Output()
	n := strings.TrimSpace(string(name))
	e := strings.TrimSpace(string(email))
	if n != "" && e != "" {
		return n + " <" + e + ">"
	}
	if n != "" {
		return n
	}
	return ""
}
