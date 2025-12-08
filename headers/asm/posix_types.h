/* Minimal shim for <asm/posix_types.h>
 * Provides a small set of POSIX-style kernel typedefs used by some
 * older Linux v6.* trees when compiling host tools on macOS.
 *
 * This is intentionally conservative — it supplies commonly-used
 * __kernel_* typedefs mapped to fixed-width stdint types appropriate
 * for 64-bit hosts. If you target 32-bit hosts or encounter additional
 * missing types, expand this shim accordingly.
 */

#ifndef _ASM_POSIX_TYPES_H
#define _ASM_POSIX_TYPES_H

#include <stdint.h>

/* Device and inode numbers — 64-bit on modern hosts */
typedef uint64_t __kernel_dev_t;
typedef uint64_t __kernel_ino_t;

/* File mode, link count, uid/gid */
typedef uint32_t __kernel_mode_t;
typedef uint32_t __kernel_nlink_t;
typedef uint32_t __kernel_uid_t;
typedef uint32_t __kernel_gid_t;

/* Offsets and process ids */
typedef int64_t  __kernel_off_t;
typedef int32_t  __kernel_pid_t;

/* Block device address */
typedef int32_t  __kernel_daddr_t;

#endif /* _ASM_POSIX_TYPES_H */
