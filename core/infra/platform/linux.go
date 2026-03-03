// Package platform provides OS-specific abstractions for ELMOS operations.
// This file implements the Linux platform with auto-detected package manager
// (apt/dnf/apk/pacman) and losetup-based disk image management.
package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NguyenTrongPhuc552003/elmos/core/infra/executor"
)

// linuxFamily identifies the Linux package manager family.
type linuxFamily int

const (
	familyDebian  linuxFamily = iota // apt-get (Ubuntu, Debian, Raspbian, etc.)
	familyFedora                     // dnf (Fedora, RHEL, CentOS Stream)
	familyAlpine                     // apk (Alpine Linux)
	familyArch                       // pacman (Arch, Manjaro)
	familyGeneric                    // Unknown — best-effort fallback
)

// linuxPlatform implements Platform for Linux hosts.
type linuxPlatform struct {
	exec     executor.Executor
	family   linuxFamily
	orbstack bool // true when running inside an OrbStack VM on macOS
	diskImg  *linuxDiskImage
	packages *linuxPackages
	paths    *linuxPaths
}

func newLinuxPlatform(exec executor.Executor) *linuxPlatform {
	if exec == nil {
		exec = executor.NewShellExecutor()
	}
	family := detectLinuxFamily()
	return &linuxPlatform{
		exec:     exec,
		family:   family,
		diskImg:  &linuxDiskImage{exec: exec},
		packages: &linuxPackages{exec: exec, family: family},
		paths:    &linuxPaths{},
	}
}

func (p *linuxPlatform) Name() string {
	var name string
	switch p.family {
	case familyDebian:
		name = "linux-debian"
	case familyFedora:
		name = "linux-fedora"
	case familyAlpine:
		name = "linux-alpine"
	case familyArch:
		name = "linux-arch"
	default:
		name = "linux"
	}
	if p.orbstack {
		name += "-orbstack"
	}
	return name
}
func (p *linuxPlatform) DiskImage() DiskImageManager { return p.diskImg }
func (p *linuxPlatform) Packages() PackageManager    { return p.packages }
func (p *linuxPlatform) Paths() PathProvider         { return p.paths }

func (p *linuxPlatform) SetExecutor(exec executor.Executor) {
	p.exec = exec
	p.diskImg.exec = exec
	p.packages.exec = exec
}

// detectLinuxFamily identifies the distro package manager by checking binaries.
func detectLinuxFamily() linuxFamily {
	switch {
	case binaryExists("apt-get"):
		return familyDebian
	case binaryExists("dnf"):
		return familyFedora
	case binaryExists("apk"):
		return familyAlpine
	case binaryExists("pacman"):
		return familyArch
	default:
		return familyGeneric
	}
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// --- DiskImageManager (Linux losetup + mount) ---

type linuxDiskImage struct{ exec executor.Executor }

// Create allocates a sparse disk image and formats it as ext4.
// It uses `truncate -s` to create a sparse file (0 actual bytes on disk, grows
// on demand) rather than `fallocate` which pre-allocates the full size.
// This matches macOS's `-type SPARSE` behavior and avoids "No space left"
// errors on tmpfs or small disks (e.g. inside OrbStack VMs).
// TODO(platform): add partition table (GPT) support in Phase 4.1.
func (d *linuxDiskImage) Create(ctx context.Context, imagePath string, sizeGB int) error {
	// Skip if the image already exists (matches darwin behaviour).
	if _, err := os.Stat(imagePath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		return fmt.Errorf("linux disk image: create parent dir: %w", err)
	}
	// Create a sparse file — only metadata is written; actual blocks are
	// allocated lazily as the filesystem inside is written to.
	sizeArg := fmt.Sprintf("%dG", sizeGB)
	if err := d.exec.Run(ctx, "truncate", "-s", sizeArg, imagePath); err != nil {
		return fmt.Errorf("linux disk image: create sparse file: %w", err)
	}
	// Format the image with ext4 so it can be immediately mounted.
	if err := d.exec.Run(ctx, "mkfs.ext4", "-F", imagePath); err != nil {
		return fmt.Errorf("linux disk image: format ext4: %w", err)
	}
	return nil
}

// Mount attaches the image via losetup and mounts it at /mnt/<name>.
// On Linux, losetup/mount/mkdir under /mnt require root privileges, so sudo
// is used unconditionally. The caller does NOT need to invoke elmos with sudo.
func (d *linuxDiskImage) Mount(ctx context.Context, imagePath string) (string, error) {
	// Allocate a loop device (requires root)
	out, err := d.exec.Output(ctx, "sudo", "losetup", "--find", "--show", imagePath)
	if err != nil {
		return "", fmt.Errorf("losetup --find: %w (hint: ensure sudo is available)", err)
	}
	loopDev := strings.TrimSpace(string(out))

	// Derive mount point from image basename: /mnt/<name>
	name := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	mountPoint := fmt.Sprintf("/mnt/%s", name)

	// Create mount point directory (requires root since /mnt is root-owned)
	if err := d.exec.Run(ctx, "sudo", "mkdir", "-p", mountPoint); err != nil {
		return "", fmt.Errorf("failed to create mount point %s: %w", mountPoint, err)
	}

	// Mount the loop device (requires root)
	if err := d.exec.Run(ctx, "sudo", "mount", loopDev, mountPoint); err != nil {
		return "", fmt.Errorf("mount %s -> %s: %w", loopDev, mountPoint, err)
	}

	// Make the mount point writable by the current user
	user := os.Getenv("USER")
	if user != "" {
		_ = d.exec.Run(ctx, "sudo", "chown", user+":"+user, mountPoint)
	}

	return mountPoint, nil
}

// Unmount unmounts and releases the loop device for mountPoint, then removes
// the mount-point directory so it doesn't linger after the workspace exits.
func (d *linuxDiskImage) Unmount(ctx context.Context, mountPoint string) error {
	// Find the loop device before unmounting so we can release it afterwards.
	loopDev := d.findLoopDevice(mountPoint)

	if err := d.exec.Run(ctx, "sudo", "umount", mountPoint); err != nil {
		return fmt.Errorf("umount %s: %w", mountPoint, err)
	}

	// Release the loop device (best-effort; ignore errors).
	if loopDev != "" {
		_ = d.exec.Run(ctx, "sudo", "losetup", "-d", loopDev)
	}

	// Remove the mount-point directory so it doesn't persist after exit.
	// Use rmdir (not rm -rf) so only empty directories are removed.
	_ = d.exec.Run(ctx, "sudo", "rmdir", mountPoint)

	return nil
}

// findLoopDevice reads /proc/mounts to find the loop device backing mountPoint.
func (d *linuxDiskImage) findLoopDevice(mountPoint string) string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == mountPoint &&
			strings.HasPrefix(fields[0], "/dev/loop") {
			return fields[0]
		}
	}
	return ""
}

// IsMounted checks /proc/mounts for a mount under /mnt/<name>.
func (d *linuxDiskImage) IsMounted(_ context.Context, name string) (bool, string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false, "", fmt.Errorf("read /proc/mounts: %w", err)
	}
	target := fmt.Sprintf("/mnt/%s", name)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == target {
			return true, target, nil
		}
	}
	return false, "", nil
}

// --- PackageManager (apt/dnf/apk/pacman) ---

// linuxPackageMap maps Homebrew/canonical formula names to distro-specific package names.
// Entries without a mapping for a given family fall through to the canonical name.
var linuxPackageMap = map[string]map[linuxFamily]string{
	// --- Cross-compiler packages ---
	"aarch64-elf-gcc":   {familyDebian: "gcc-aarch64-linux-gnu", familyFedora: "gcc-aarch64-linux-gnu", familyAlpine: "aarch64-linux-musl-cross"},
	"arm-none-eabi-gcc": {familyDebian: "gcc-arm-none-eabi", familyFedora: "arm-none-eabi-gcc", familyAlpine: "arm-none-eabi-gcc"},

	// --- Board support packages ---
	"dtc":          {familyDebian: "device-tree-compiler", familyFedora: "dtc", familyAlpine: "dtc"},
	"qemu":         {familyDebian: "qemu-system-arm", familyFedora: "qemu-system-arm", familyAlpine: "qemu-system-arm"},
	"u-boot-tools": {familyDebian: "u-boot-tools", familyFedora: "uboot-tools", familyAlpine: "u-boot-tools"},
	"debootstrap":  {familyDebian: "debootstrap", familyFedora: "debootstrap", familyAlpine: "debootstrap"},
	"e2fsprogs":    {familyDebian: "e2fsprogs", familyFedora: "e2fsprogs", familyAlpine: "e2fsprogs"},

	// --- Build Tools (from RequiredPackages) ---
	"llvm":      {familyDebian: "clang", familyFedora: "clang", familyAlpine: "clang", familyArch: "clang"},
	"lld":       {familyDebian: "lld", familyFedora: "lld", familyAlpine: "lld", familyArch: "lld"},
	"gnu-sed":   {familyDebian: "sed", familyFedora: "sed", familyAlpine: "sed", familyArch: "sed"},
	"make":      {familyDebian: "make", familyFedora: "make", familyAlpine: "make", familyArch: "make"},
	"libelf":    {familyDebian: "libelf-dev", familyFedora: "elfutils-libelf-devel", familyAlpine: "elfutils-dev", familyArch: "libelf"},
	"git":       {familyDebian: "git", familyFedora: "git", familyAlpine: "git", familyArch: "git"},
	"fakeroot":  {familyDebian: "fakeroot", familyFedora: "fakeroot", familyAlpine: "fakeroot", familyArch: "fakeroot"},
	"wget":      {familyDebian: "wget", familyFedora: "wget", familyAlpine: "wget", familyArch: "wget"},
	"coreutils": {familyDebian: "coreutils", familyFedora: "coreutils", familyAlpine: "coreutils", familyArch: "coreutils"},
	"go":        {familyDebian: "golang", familyFedora: "golang", familyAlpine: "go", familyArch: "go"},
	"go-task":   {}, // Not in distro repos; installed via https://taskfile.dev

	// --- Toolchain Dependencies (from RequiredPackages) ---
	"binutils": {familyDebian: "binutils", familyFedora: "binutils", familyAlpine: "binutils", familyArch: "binutils"},
	"gcc":      {familyDebian: "gcc", familyFedora: "gcc", familyAlpine: "gcc", familyArch: "gcc"},
	"gmp":      {familyDebian: "libgmp-dev", familyFedora: "gmp-devel", familyAlpine: "gmp-dev", familyArch: "gmp"},
	"mpfr":     {familyDebian: "libmpfr-dev", familyFedora: "mpfr-devel", familyAlpine: "mpfr-dev", familyArch: "mpfr"},
	"libmpc":   {familyDebian: "libmpc-dev", familyFedora: "libmpc-devel", familyAlpine: "mpc1-dev", familyArch: "libmpc"},
	"isl":      {familyDebian: "libisl-dev", familyFedora: "isl-devel", familyAlpine: "isl-dev", familyArch: "isl"},
	"texinfo":  {familyDebian: "texinfo", familyFedora: "texinfo", familyAlpine: "texinfo", familyArch: "texinfo"},
	"bison":    {familyDebian: "bison", familyFedora: "bison", familyAlpine: "bison", familyArch: "bison"},
	"gawk":     {familyDebian: "gawk", familyFedora: "gawk", familyAlpine: "gawk", familyArch: "gawk"},
	"autoconf": {familyDebian: "autoconf", familyFedora: "autoconf", familyAlpine: "autoconf", familyArch: "autoconf"},
	"automake": {familyDebian: "automake", familyFedora: "automake", familyAlpine: "automake", familyArch: "automake"},
	"libtool":  {familyDebian: "libtool", familyFedora: "libtool", familyAlpine: "libtool", familyArch: "libtool"},
	"ncurses":  {familyDebian: "libncurses-dev", familyFedora: "ncurses-devel", familyAlpine: "ncurses-dev", familyArch: "ncurses"},
	"xz":       {familyDebian: "xz-utils", familyFedora: "xz", familyAlpine: "xz", familyArch: "xz"},
}

type linuxPackages struct {
	exec   executor.Executor
	family linuxFamily
}

// resolvePackageName translates a canonical package name to the distro-specific name.
// Returns "" if the package map entry exists but has no mapping for this family
// (e.g. "go-task" which is not in any distro repo).
func (p *linuxPackages) resolvePackageName(pkg string) string {
	distroMap, ok := linuxPackageMap[pkg]
	if !ok {
		return pkg // not in map at all — use as-is
	}
	if len(distroMap) == 0 {
		return "" // explicit empty map — no distro package
	}
	if name, ok := distroMap[p.family]; ok {
		return name
	}
	return pkg // family not in map — fall back to canonical name
}

func (p *linuxPackages) IsInstalled(pkg string) bool {
	native := p.resolvePackageName(pkg)
	// Empty mapping means no distro package exists; fall back to binary check.
	if native == "" {
		_, err := exec.LookPath(pkg)
		return err == nil
	}
	return p.isSinglePackageInstalled(native)
}

// isSinglePackageInstalled checks a single distro-native package name.
func (p *linuxPackages) isSinglePackageInstalled(native string) bool {
	switch p.family {
	case familyDebian:
		out, _ := p.exec.Output(context.Background(), "dpkg", "-s", native)
		return strings.Contains(string(out), "Status: install ok installed")
	case familyFedora:
		err := p.exec.Run(context.Background(), "rpm", "-q", native)
		return err == nil
	case familyAlpine:
		err := p.exec.Run(context.Background(), "apk", "info", "-e", native)
		return err == nil
	case familyArch:
		err := p.exec.Run(context.Background(), "pacman", "-Q", native)
		return err == nil
	default:
		_, err := exec.LookPath(native)
		return err == nil
	}
}

func (p *linuxPackages) Install(ctx context.Context, pkg string) error {
	native := p.resolvePackageName(pkg)
	if native == "" {
		return fmt.Errorf("package %q has no distro mapping; install manually", pkg)
	}
	switch p.family {
	case familyDebian:
		return p.exec.Run(ctx, "apt-get", "install", "-y", native)
	case familyFedora:
		return p.exec.Run(ctx, "dnf", "install", "-y", native)
	case familyAlpine:
		return p.exec.Run(ctx, "apk", "add", "--no-cache", native)
	case familyArch:
		return p.exec.Run(ctx, "pacman", "-S", "--noconfirm", native)
	default:
		return fmt.Errorf("package install not supported on this Linux variant")
	}
}

func (p *linuxPackages) ListInstalled() ([]string, error) {
	var out []byte
	var err error
	switch p.family {
	case familyDebian:
		out, err = p.exec.Output(context.Background(), "dpkg", "--get-selections")
	case familyFedora:
		out, err = p.exec.Output(context.Background(), "rpm", "-qa", "--queryformat", "%{NAME}\n")
	case familyAlpine:
		out, err = p.exec.Output(context.Background(), "apk", "list", "--installed")
	default:
		return nil, fmt.Errorf("listing packages not supported on this Linux variant")
	}
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			result = append(result, strings.Fields(line)[0])
		}
	}
	return result, nil
}

// GetBinPath searches for the binary in standard system locations.
func (p *linuxPackages) GetBinPath(pkg string) string {
	native := p.resolvePackageName(pkg)
	if path, err := exec.LookPath(native); err == nil {
		return filepath.Dir(path)
	}
	return ""
}

func (p *linuxPackages) GetLibPath(pkg string) string {
	for _, candidate := range []string{"/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/lib/aarch64-linux-gnu"} {
		if _, err := os.Stat(filepath.Join(candidate, pkg)); err == nil {
			return candidate
		}
	}
	return "/usr/lib"
}

func (p *linuxPackages) GetIncludePath(_ string) string {
	return "/usr/include"
}

// --- PathProvider (Linux) ---

type linuxPaths struct{}

// WorkspaceRoot returns /mnt/<name> as the workspace mount point.
func (p *linuxPaths) WorkspaceRoot(name string) string {
	return fmt.Sprintf("/mnt/%s", name)
}

func (p *linuxPaths) CacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".elmos")
	}
	return filepath.Join(homeDir, ".elmos")
}

func (p *linuxPaths) ToolchainDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", "x-tools")
	}
	return filepath.Join(homeDir, "x-tools")
}
