# The object file to build
obj-m += char-test.o

# KDIR is the path to the compiled kernel source on your sparse image
# We use ?= so that module.sh can override this path
KDIR ?= ${KERNEL_DIR}
PWD  := $(shell pwd)

# Default 'all' target calls the Kernel's build system (Kbuild)
all:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

# Clean up build artifacts
clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean
