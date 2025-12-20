#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

// Metadata for the -f (info) feature in module.sh
MODULE_LICENSE("GPL");
MODULE_AUTHOR("NguyenTrongPhuc");
MODULE_DESCRIPTION("A simple Hello World module for MacOS-Linux Dev");
MODULE_VERSION("1.0");

static int __init hello_init(void) {
	printk(KERN_INFO "  [HELLO] Module loaded successfully!\n");
	printk(KERN_INFO "  [HELLO] Hello from the macOS-built kernel module!\n");
	return 0;
}

static void __exit hello_exit(void) {
	printk(KERN_INFO "  [HELLO] Module unloaded. Goodbye!\n");
}

module_init(hello_init);
module_exit(hello_exit);
