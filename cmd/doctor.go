// Package cmd implements the Cobra CLI commands for elmos.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/NguyenTrongPhuc552003/elmos/internal/core"
)

// doctorCmd - check environment and dependencies
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and dependencies",
	Long:  `Verify that all required tools and dependencies are installed correctly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

// RequiredPackage represents a Homebrew package dependency
type RequiredPackage struct {
	Name        string
	Description string
	Required    bool
}

// requiredPackages lists all Homebrew dependencies
var requiredPackages = []RequiredPackage{
	{"llvm", "LLVM/Clang toolchain", true},
	{"lld", "LLVM linker", true},
	{"gnu-sed", "GNU sed (kernel requires it)", true},
	{"make", "GNU make 4.0+", true},
	{"libelf", "ELF library", true},
	{"git", "Git version control", true},
	{"qemu", "QEMU emulator", true},
	{"fakeroot", "Fake root for packaging", true},
	{"e2fsprogs", "ext4 filesystem tools", true},
	{"wget", "File downloader", false},
	{"coreutils", "GNU core utilities", true},
}

// requiredTaps lists required Homebrew taps
var requiredTaps = []string{
	"messense/macos-cross-toolchains",
}

// requiredHeaders lists header files that should exist
var requiredHeaders = []string{
	"elf.h",
	"byteswap.h",
}

func runDoctor() error {
	fmt.Println()
	printInfo("ELMOS Doctor - Environment Check")
	fmt.Println()

	issuesFound := 0

	// Check Homebrew
	printStep("Checking Homebrew...")
	if !checkCommandExists("brew") {
		printError("Homebrew not found")
		printInfo("Install from: https://brew.sh")
		issuesFound++
	} else {
		printSuccess("Homebrew found")
	}
	fmt.Println()

	// Check taps
	printStep("Checking Homebrew taps...")
	issuesFound += checkTaps()
	fmt.Println()

	// Check packages
	printStep("Checking required packages...")
	issuesFound += checkPackages()
	fmt.Println()

	// Check headers
	printStep("Checking custom headers...")
	issuesFound += checkHeaders()
	fmt.Println()

	// Check architecture-specific GDB
	printStep("Checking cross-debuggers...")
	checkCrossGDB()
	fmt.Println()

	// Check architecture-specific GCC
	printStep("Checking cross-compilers...")
	checkCrossGCC()
	fmt.Println()

	// Summary
	if issuesFound == 0 {
		printSuccess("All checks passed! Environment ready for kernel development.")
		return nil
	}

	printWarn("Found %d issue(s). Please fix before proceeding.", issuesFound)
	return fmt.Errorf("doctor found %d issues", issuesFound)
}

func checkCommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func checkTaps() int {
	issues := 0
	out, err := exec.Command("brew", "tap").Output()
	if err != nil {
		printError("Failed to list Homebrew taps")
		return 1
	}

	installedTaps := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, tap := range requiredTaps {
		found := false
		for _, installed := range installedTaps {
			if installed == tap {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("  ✓ %s\n", tap)
		} else {
			fmt.Printf("  ✗ %s (missing)\n", tap)
			printInfo("  Fix: brew tap %s", tap)
			issues++
		}
	}

	return issues
}

func checkPackages() int {
	issues := 0
	out, err := exec.Command("brew", "list", "--formulae").Output()
	if err != nil {
		printError("Failed to list Homebrew packages")
		return 1
	}

	installedPkgs := strings.Split(strings.TrimSpace(string(out)), "\n")
	installedSet := make(map[string]bool)
	for _, pkg := range installedPkgs {
		installedSet[pkg] = true
	}

	var missing []string

	for _, pkg := range requiredPackages {
		if installedSet[pkg.Name] {
			fmt.Printf("  ✓ %s\n", pkg.Name)
		} else {
			status := "missing"
			if !pkg.Required {
				status = "optional, missing"
			}
			fmt.Printf("  ✗ %s (%s)\n", pkg.Name, status)
			if pkg.Required {
				missing = append(missing, pkg.Name)
				issues++
			}
		}
	}

	if len(missing) > 0 {
		printInfo("Fix: brew install %s", strings.Join(missing, " "))
	}

	return issues
}

func checkHeaders() int {
	cfg := core.GetConfig()
	headersDir := cfg.Paths.LibrariesDir
	issues := 0

	// Check if headers directory exists
	if _, err := os.Stat(headersDir); os.IsNotExist(err) {
		printError("Headers directory not found: %s", headersDir)
		return 1
	}

	for _, header := range requiredHeaders {
		headerPath := filepath.Join(headersDir, header)
		if _, err := os.Stat(headerPath); err == nil {
			fmt.Printf("  ✓ %s\n", header)
		} else {
			fmt.Printf("  ✗ %s (missing)\n", header)
			issues++
		}
	}

	// Check asm symlinks
	asmDir := filepath.Join(headersDir, "asm")
	if info, err := os.Stat(asmDir); err == nil && info.IsDir() {
		fmt.Printf("  ✓ asm/ (directory)\n")
	} else {
		fmt.Printf("  ✗ asm/ (missing)\n")
		issues++
	}

	return issues
}

func checkCrossGDB() {
	gdbs := []struct {
		arch string
		bin  string
	}{
		{"RISC-V", "riscv64-elf-gdb"},
		{"ARM64", "aarch64-elf-gdb"},
		{"ARM32", "arm-none-eabi-gdb"},
	}

	for _, gdb := range gdbs {
		if checkCommandExists(gdb.bin) {
			fmt.Printf("  ✓ %s (%s)\n", gdb.arch, gdb.bin)
		} else {
			fmt.Printf("  ○ %s (%s) - optional\n", gdb.arch, gdb.bin)
		}
	}
}

func checkCrossGCC() {
	gccs := []struct {
		arch string
		bin  string
	}{
		{"RISC-V", "riscv64-elf-gcc"},
		{"ARM64", "aarch64-elf-gcc"},
		{"ARM32", "arm-none-eabi-gcc"},
	}

	for _, gcc := range gccs {
		if checkCommandExists(gcc.bin) {
			fmt.Printf("  ✓ %s (%s)\n", gcc.arch, gcc.bin)
		} else {
			fmt.Printf("  ○ %s (%s) - optional\n", gcc.arch, gcc.bin)
		}
	}
}
