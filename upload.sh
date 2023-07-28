#!/bin/bash

set -euo pipefail

if [[ -z "${GRAFANA_API_KEY}" ]]; then
  echo "GRAFANA_API_KEY is not set"
  exit 1
fi

rm -rf dist grafana-llm-app
npm run build
mage -v
npx @grafana/sign-plugin@latest
mv dist grafana-llm-app
zip grafana-llm-app-0.1.0.zip grafana-llm-app -r
gsutil cp ./grafana-llm-app-0.1.0.zip gs://grafana-llm-app/
