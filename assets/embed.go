// Package assets provides embedded template files for elmos.
package assets

import "embed"

//go:embed templates/*
var Templates embed.FS

// GetModuleTemplate returns the module source template.
func GetModuleTemplate() ([]byte, error) {
	return Templates.ReadFile("templates/module/module.c.tmpl")
}

// GetModuleMakefile returns the module Makefile template.
func GetModuleMakefile() ([]byte, error) {
	return Templates.ReadFile("templates/module/Makefile.tmpl")
}

// GetAppTemplate returns the app source template.
func GetAppTemplate() ([]byte, error) {
	return Templates.ReadFile("templates/app/main.c.tmpl")
}

// GetAppMakefile returns the app Makefile template.
func GetAppMakefile() ([]byte, error) {
	return Templates.ReadFile("templates/app/Makefile.tmpl")
}

// GetInitScript returns the init script template.
func GetInitScript() ([]byte, error) {
	return Templates.ReadFile("templates/init/init.sh.tmpl")
}

// GetGuestSync returns the guest sync script template.
func GetGuestSync() ([]byte, error) {
	return Templates.ReadFile("templates/init/guesync.sh.tmpl")
}
