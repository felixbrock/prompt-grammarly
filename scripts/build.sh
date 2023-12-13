#!/bin/bash

echo "Transpiling Tailwind CSS..."
bunx tailwindcss -i ./static/index.css -o ./static/index_transpiled.css

if [ $? -eq 0 ]; then
    echo "Transpiling Templ components..."
    ~/go/bin/templ generate -path ./internal/component
    echo "Success"
else
    echo "Failed to transpile Tailwind CSS. Aborting..."
    exit 1
fi
