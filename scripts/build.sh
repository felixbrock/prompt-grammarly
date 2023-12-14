#!/bin/bash

bunx tailwindcss -i ./static/index.css -o ./static/index_transpiled.css
echo "Transpiled tailwind css"

if [ $? -eq 0 ]; then
    ~/go/bin/templ generate -path ./internal/component
    echo "Transpiled templ components"
else
    echo "Failed to transpile Tailwind CSS. Aborting..."
    exit 1
fi

go build -o ./tmp/main .
echo "Built go binary"
