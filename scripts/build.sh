#!/bin/bash

# Running bunx tailwindcss command
bunx tailwindcss -i ./static/index.css -o ./static/index_transpiled.css

# Check if the previous command was successful
if [ $? -eq 0 ]; then
    # Running templ generate command
    ~/go/bin/templ generate -path ./internal/components
else
    echo "Failed to transpile Tailwind CSS. Aborting..."
    exit 1
fi
