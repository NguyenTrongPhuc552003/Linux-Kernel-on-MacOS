// SPDX-License-Identifier: GPL-2.0
/*
 * hello-world - Kernel module
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

static int __init hello_world_init(void)
{
    pr_info("hello-world: Module loaded\n");
    return 0;
}

static void __exit hello_world_exit(void)
{
    pr_info("hello-world: Module unloaded\n");
}

module_init(hello_world_init);
module_exit(hello_world_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Your Name");
MODULE_DESCRIPTION("A simple kernel module");
MODULE_VERSION("1.0");
