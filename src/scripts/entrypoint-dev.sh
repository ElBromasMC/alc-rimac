#!/bin/sh
# Check if "node_modules" exists in the current directory.
if [ -d "node_modules" ]; then
    echo "Found node_modules in the current directory. Removing it..."
    rm -rf "node_modules"
fi

# Check if "node_modules" exists in the parent directory.
if [ -d "../node_modules" ]; then
    echo "Found node_modules in the parent directory. Moving it to the current directory..."
    mv "../node_modules" .
else
    echo "Error: node_modules folder not found in the parent directory." >&2
    exit 1
fi

# Clean build artifacts and run the live reload!
make clean
exec /bin/bash

