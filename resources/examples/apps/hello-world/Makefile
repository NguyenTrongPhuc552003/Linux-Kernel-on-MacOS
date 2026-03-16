# 1. Variables: Define output name and settings here
TARGET  := hello_world
CC      ?= clang
CFLAGS  ?= -Wall -Wextra
LDFLAGS ?= -static

# 2. Dynamic Sources: Automatically find all .c files in this directory
SRCS    := $(wildcard *.c)

# 3. Objects: Automatically derive .o filenames from .c filenames
OBJS    := $(SRCS:.c=.o)

# 4. Default Rule
all: $(TARGET)

# 5. Link Step: Create the binary from object files
$(TARGET): $(OBJS)
	$(CC) $(OBJS) -o $@ $(LDFLAGS)

# 6. Compile Step: generic rule to turn any .c into .o
%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

# 7. Clean Up
clean:
	rm -f $(TARGET) $(OBJS)

.PHONY: all clean
