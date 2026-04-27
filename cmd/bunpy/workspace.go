package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tamnd/bunpy/v1/pkg/workspace"
)

// findWorkspaceRoot is the cmd-layer wrapper around workspace.FindRoot.
// Returns ("", nil) when no workspace is found (ErrNoWorkspace is not
// an error at the call sites that want to fall through to single-project
// mode).
func findWorkspaceRoot(cwd string) (string, error) {
	root, err := workspace.FindRoot(cwd)
	if err != nil {
		return "", nil
	}
	return root, nil
}

// wsHandle is a thin cmd-layer handle around workspace.Workspace.
type wsHandle struct {
	ws *workspace.Workspace
}

func loadWorkspace(root string) (*wsHandle, error) {
	ws, err := workspace.Load(root)
	if err != nil {
		return nil, err
	}
	return &wsHandle{ws: ws}, nil
}

func (h *wsHandle) findMember(name string) (workspace.Member, bool) {
	return workspace.MemberByName(h.ws, name)
}

func workspaceSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	list := false
	wsRoot := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("workspace", stdout, stderr)
		case "--list":
			list = true
		case "--workspace":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy workspace: --workspace requires a value")
			}
			i++
			wsRoot = args[i]
		default:
			return 1, fmt.Errorf("bunpy workspace: unknown flag %q", a)
		}
	}

	if wsRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 1, fmt.Errorf("bunpy workspace: %w", err)
		}
		found, err := workspace.FindRoot(cwd)
		if err != nil {
			return 1, fmt.Errorf("bunpy workspace: %w", err)
		}
		wsRoot = found
	}

	ws, err := workspace.Load(wsRoot)
	if err != nil {
		return 1, fmt.Errorf("bunpy workspace: %w", err)
	}

	if list || len(args) == 0 {
		for _, m := range ws.Members {
			rel, err := filepath.Rel(ws.Root, m.Path)
			if err != nil {
				rel = m.Path
			}
			fmt.Fprintf(stdout, "%-20s %s\n", m.Name, rel)
		}
		return 0, nil
	}

	return 1, fmt.Errorf("bunpy workspace: no operation specified (try --list)")
}
