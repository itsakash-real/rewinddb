//go:build ignore
// +build ignore

// Basic Nimbi SDK usage example.
// Run from the repo root:
//
//	go run examples/basic/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsakash-real/nimbi/sdk"
)

func main() {
	// ── 1. Set up a temporary project directory ───────────────────────────────
	projectDir, err := os.MkdirTemp("", "nimbi-example-*")
	must(err)
	defer os.RemoveAll(projectDir)

	fmt.Printf("Project root: %s\n\n", projectDir)

	// Seed some files
	writeFile(projectDir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("Hello, Nimbi!")
}`)
	writeFile(projectDir, "go.mod", "module example\n\ngo 1.22\n")
	writeFile(projectDir, "README.md", "# My Project\n")

	// ── 2. Initialise a Nimbi repository ───────────────────────────────────
	fmt.Println("── Initializing repository ──────────────────────────────")
	client, err := sdk.Init(projectDir)
	must(err)
	fmt.Printf("✓ Repository initialized at %s\n\n", filepath.Join(projectDir, ".rewind"))

	// ── 3. Save three checkpoints ─────────────────────────────────────────────
	fmt.Println("── Saving checkpoints ───────────────────────────────────")

	cp1, err := client.Save("initial project setup")
	must(err)
	fmt.Printf("✓ [1] %s  %q\n", cp1.ID[:8], cp1.Message)

	writeFile(projectDir, "main.go", `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Printf("Hello, %s!\n", os.Args[1])
	} else {
		fmt.Println("Hello, Nimbi!")
	}
}`)
	writeFile(projectDir, "internal/greet.go", `package internal

func Greet(name string) string {
	return "Hello, " + name + "!"
}`)

	cp2, err := client.SaveWithTags("add greeting support", []string{"v0.1"})
	must(err)
	fmt.Printf("✓ [2] %s  %q  [tags: v0.1]\n", cp2.ID[:8], cp2.Message)

	writeFile(projectDir, "main.go", `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	name := "World"
	if len(os.Args) > 1 {
		name = strings.Join(os.Args[1:], " ")
	}
	fmt.Printf("Hello, %s!\n", name)
}`)

	cp3, err := client.SaveWithTags("improve arg handling", []string{"v0.2"})
	must(err)
	fmt.Printf("✓ [3] %s  %q  [tags: v0.2]\n\n", cp3.ID[:8], cp3.Message)

	// ── 4. Modify a file (without saving) ────────────────────────────────────
	fmt.Println("── Modifying file without saving ────────────────────────")
	writeFile(projectDir, "main.go", `package main // WORK IN PROGRESS`)
	writeFile(projectDir, "debug.log", "debug output here")
	fmt.Println("✓ main.go modified, debug.log added (not saved yet)\n")

	// ── 5. Show status ────────────────────────────────────────────────────────
	fmt.Println("── Status ───────────────────────────────────────────────")
	status, err := client.Status()
	must(err)
	fmt.Printf("Branch:  %s\n", status.CurrentBranch.Name)
	fmt.Printf("HEAD:    %s  %q\n", status.HeadCheckpoint.ID[:8], status.HeadCheckpoint.Message)
	if !status.IsClean {
		fmt.Println("Pending changes:")
		for _, f := range status.ModifiedFiles {
			fmt.Printf("  [~] %s\n", f)
		}
		for _, f := range status.AddedFiles {
			fmt.Printf("  [+] %s\n", f)
		}
	}
	fmt.Printf("Storage: %d objects\n\n", status.StorageStats.ObjectCount)

	// ── 6. Goto first checkpoint ─────────────────────────────────────────────
	fmt.Println("── Going to checkpoint 1 ────────────────────────────────")
	restored, err := client.Goto(cp1.ID[:8])
	must(err)
	fmt.Printf("✓ Restored to %s  %q\n", restored.ID[:8], restored.Message)
	data, _ := os.ReadFile(filepath.Join(projectDir, "main.go"))
	fmt.Printf("  main.go first line: %q\n\n", firstLine(string(data)))

	// ── 7. Diff cp1 → cp3 ────────────────────────────────────────────────────
	fmt.Println("── Diff cp1 → cp3 ───────────────────────────────────────")
	diffResult, err := client.Diff(cp1.ID, cp3.ID)
	must(err)

	diffEngine := sdk.NewDiffEngine(client)
	fmt.Println(diffEngine.Summary(diffResult))
	fmt.Print(diffEngine.PrettyPrint(diffResult))

	// ── 8. List all checkpoints ───────────────────────────────────────────────
	fmt.Println("\n── Checkpoint history ───────────────────────────────────")
	cps, err := client.List(sdk.ListOpts{})
	must(err)
	for _, cp := range cps {
		fmt.Printf("  %s  %q\n", cp.ID[:8], cp.Message)
	}

	// ── 9. GC dry run ─────────────────────────────────────────────────────────
	fmt.Println("\n── GC (dry run) ─────────────────────────────────────────")
	gcResult, err := client.GC(true)
	must(err)
	fmt.Printf("Would free: %d objects, %d bytes\n", gcResult.RemovedObjects, gcResult.FreedBytes)

	fmt.Println("\nDone.")
}

func writeFile(root, rel, content string) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(abs), 0o755)
	os.WriteFile(abs, []byte(content), 0o644)
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
