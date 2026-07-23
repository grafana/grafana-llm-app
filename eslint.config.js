import { defineConfig } from "eslint/config";
import grafanaConfig from "@grafana/eslint-config";

export default defineConfig([
    // Global ignores - migrated from .eslintignore
    {
        ignores: [
            // Dependencies
            "**/node_modules/",
            
            // Build outputs
            "**/dist/",
            "**/build/",
            "**/coverage/",
            "**/artifacts/",
            "**/work/",
            "**/ci/",
            "**/*.tsbuildinfo",
            
            // Test artifacts
            "**/test-results/",
            "**/playwright-report/",
            "**/e2e-results/",
            "**/cypress/videos",
            "**/cypress/report.json",
            
            // Package files
            "**/package-lock.json",
            "**/yarn.lock",
            "**/pnpm-lock.yaml",
            
            // Logs
            "**/*.log",
            "**/npm-debug.log*",
            "**/yarn-debug.log*",
            "**/yarn-error.log*",
            "**/.pnpm-debug.log*",
            
            // IDE
            "**/.vscode/",
            "**/.idea/",
            "**/*.swp",
            "**/*.swo",
            "**/*~",
            
            // OS
            "**/.DS_Store",
            "**/Thumbs.db",
            
            // Generated files
            "**/*.generated.*",
            "**/__to-upload__",
            
            // Configuration files that are typically not linted
            "**/*.config.js",
            "**/webpack.config.*",
            "**/rollup.config.*",
            "**/jest.config.*",
            "**/playwright.config.*",
            
            // Cache files
            "**/.eslintcache",
            "**/.cache/",
        ],
    },
    
    // Base configuration for all JS/TS files
    ...grafanaConfig,
]);

