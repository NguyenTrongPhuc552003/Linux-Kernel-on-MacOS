#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/fs.h>

static int major;

static int __init char_test_init(void) {
    major = register_chrdev(0, "char_test", NULL);
    if (major < 0) {
        printk(KERN_ERR "  [CHAR] Failed to register character device\n");
        return major;
    }
    printk(KERN_INFO "  [CHAR] Module loaded with major number %d\n", major);
    return 0;
}

static void __exit char_test_exit(void) {
    unregister_chrdev(major, "char_test");
    printk(KERN_INFO "  [CHAR] Module unloaded successfully\n");
}

module_init(char_test_init);
module_exit(char_test_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("HolyShit");
MODULE_DESCRIPTION("A simple character driver to test QEMU virtio-serial communication and system calls");
MODULE_VERSION("0.1");
