#!/usr/bin/env sh
# Render every docs/diagrams/*.puml to SVG with the official PlantUML image.
set -eu
cd "$(dirname "$0")/../docs/diagrams"
docker run --rm -u "$(id -u):$(id -g)" -e JAVA_TOOL_OPTIONS=-Duser.home=/tmp \
  -v "$PWD":/data plantuml/plantuml:1.2024.7 -tsvg /data/*.puml
