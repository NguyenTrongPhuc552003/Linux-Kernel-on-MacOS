#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/netfilter.h>
#include <linux/netfilter_ipv4.h>
#include <linux/ip.h>
#include <linux/tcp.h>

static struct nf_hook_ops nfho;
static unsigned long packet_count = 0;
static unsigned long byte_count = 0;

// Hook function: called for every packet passing through the PRE_ROUTING stage
static unsigned int hook_func(void *priv, struct sk_buff *skb, const struct nf_hook_state *state) {
    struct iphdr *ip_header;

    if (!skb) return NF_ACCEPT;

    ip_header = ip_hdr(skb);
    if (ip_header) {
        packet_count++;
        byte_count += ntohs(ip_header->tot_len);

        // Log every 100 packets to avoid flooding dmesg
        if (packet_count % 100 == 0) {
            printk(KERN_INFO "  [NET-MON] Total: %lu packets, %lu bytes\n", packet_count, byte_count);
        }
    }

    return NF_ACCEPT; // Allow the packet to continue its journey
}

static int __init net_monitor_init(void) {
    nfho.hook = hook_func;
    nfho.hooknum = NF_INET_PRE_ROUTING; // Intercept incoming packets
    nfho.pf = PF_INET;                  // IPv4
    nfho.priority = NF_IP_PRI_FIRST;    // High priority

    nf_register_net_hook(&init_net, &nfho);
    printk(KERN_INFO "  [NET-MON] Traffic monitor initialized and hooked into Netfilter\n");
    return 0;
}

static void __exit net_monitor_exit(void) {
    nf_unregister_net_hook(&init_net, &nfho);
    printk(KERN_INFO "  [NET-MON] Traffic monitor stopped. Final stats: %lu packets, %lu bytes\n", packet_count, byte_count);
}

module_init(net_monitor_init);
module_exit(net_monitor_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Kernel Dev Team");
MODULE_DESCRIPTION("Real network packet monitor using Netfilter hooks to track guest traffic overhead");
MODULE_VERSION("1.2.0");
