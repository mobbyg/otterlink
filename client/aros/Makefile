# Native AROS client scaffold.
# Use a compiler/SDK that matches the target AROS distribution and ABI.

AROS_CC ?= x86_64-aros-gcc
CPPFLAGS ?= -D__AROS__
CFLAGS ?= -Wall -Wextra -O2
LDFLAGS ?=
LDLIBS ?= -lmui

TARGET := otterlink-aros
SOURCES := main.c
OBJECTS := $(SOURCES:.c=.o)

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJECTS)
	$(AROS_CC) $(LDFLAGS) -o $@ $(OBJECTS) $(LDLIBS)

%.o: %.c
	$(AROS_CC) $(CPPFLAGS) $(CFLAGS) -c -o $@ $<

clean:
	rm -f $(OBJECTS) $(TARGET)
