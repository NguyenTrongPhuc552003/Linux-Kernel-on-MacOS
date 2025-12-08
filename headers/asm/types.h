/* Minimal shim for <asm/types.h>
 * Provides common kernel-style integer typedefs required by older
 * Linux tags (v6.0 - v6.12) when compiling host tools on macOS.
 *
 * This file is intentionally small — it maps the commonly used
 * __u8/__s8 ... __u64/__s64 types to stdint.h types.
 */

#ifndef _ASM_TYPES_H
#define _ASM_TYPES_H

#include <stdint.h>

typedef uint8_t  __u8;
typedef int8_t   __s8;
typedef uint16_t __u16;
typedef int16_t  __s16;
typedef uint32_t __u32;
typedef int32_t  __s32;
typedef uint64_t __u64;
typedef int64_t  __s64;

typedef unsigned long __kernel_ulong_t;
typedef long          __kernel_long_t;

#endif /* _ASM_TYPES_H */
