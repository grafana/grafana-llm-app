# End-to-End Testing Guide

This document explains how to run end-to-end tests for the Grafana LLM App using Playwright and Docker Compose.

## Overview

The testing setup uses:
- **Playwright** for browser automation and testing
- **Docker Compose** for orchestrating Grafana and Playwright containers
- **@grafana/plugin-e2e** for Grafana-specific testing utilities
- **Visual regression testing** for UI consistency

## Prerequisites

- Docker and Docker Compose
- Node.js 22+ (for local development)
- Built plugin artifacts (`npm run build && npm run backend:build`)

## Quick Start

### Option 1: Full Automated Testing (CI-style)
```bash
# Run complete test suite (builds, starts services, tests, cleans up)
npm run test:e2e-full
```

### Option 2: Manual Testing (Development)
```bash
# Start Grafana
npm run server:detach

# Run tests in Docker container
npm run playwright:run

# Clean up
npm run server:down
```

### Option 3: Interactive Development
```bash
# Start Grafana and Playwright server
npm run test:e2e-dev

# In another terminal, run tests against the server
npm run playwright:test

# Stop everything
npm run playwright:stop && npm run server:down
```

## Available Commands

### Core Testing Commands
- `npm run test:e2e` - Full automated test run (start Grafana → run tests → cleanup)
- `npm run test:e2e-full` - Build plugin + full test run
- `npm run test:e2e-dev` - Start services for development testing

### Playwright Container Commands
- `npm run playwright:run` - Run tests in Docker container (one-shot)
- `npm run playwright:server` - Start Playwright server for interactive testing
- `npm run playwright:stop` - Stop Playwright services
- `npm run playwright:update-snapshots` - Update visual test snapshots

### Local Testing Commands
- `npm run playwright:test` - Run tests using remote Playwright server
- `npm run playwright:test-local` - Run tests locally (requires local Playwright install)

### Service Management
- `npm run server:detach` - Start Grafana in background
- `npm run server:down` - Stop Grafana and cleanup
- `npm run playwright:logs` - View Playwright server logs
- `npm run playwright:logs-runner` - View test runner logs

## Docker Compose Services

The testing setup uses Docker Compose profiles:

### Default Profile (Grafana)
```yaml
services:
  grafana:
    # Main Grafana instance with plugin mounted
```

### Testing Profile
```yaml
services:
  playwright-server:
    # Interactive Playwright server (ws://localhost:5000)
  
  playwright-runner:  
    # One-shot test runner with automatic Grafana health check
```

## Test Structure

### Test Categories

1. **Functional Tests** (`tests/example.spec.ts`)
   - Plugin loading and configuration
   - MCP functionality testing
   - API endpoint verification
   - UI element presence and behavior

2. **Visual Tests** (`tests/visual.spec.ts`)
   - Full page screenshots
   - Responsive design testing
   - Theme compatibility
   - Component-level visual regression

### Test Features

- **Multi-browser support** (Chrome, Firefox, Safari/WebKit)
- **Automatic Grafana health checking** before tests start
- **Intelligent error filtering** (ignores browser-specific errors)
- **Environment-aware testing** (handles different runtime environments)
- **Comprehensive visual regression** with automatic snapshot management

## Development Workflow

### Adding New Tests

1. Create test files in `tests/` directory
2. Use Grafana plugin test utilities:
   ```typescript
   import { test, expect } from '@grafana/plugin-e2e';
   ```

3. Add test IDs to components:
   ```typescript
   // Add to testIds.ts
   export const testIds = {
     myFeature: {
       container: 'data-testid my-feature-container',
       button: 'data-testid my-feature-button'
     }
   };
   ```

4. Write tests using the IDs:
   ```typescript
   test('my feature works', async ({ page }) => {
     await page.goto('/a/grafana-llm-app');
     await expect(page.getByTestId('my-feature-container')).toBeVisible();
   });
   ```

### Running Specific Tests

```bash
# Run only functional tests
npm run playwright:test tests/example.spec.ts

# Run only visual tests  
npm run playwright:test tests/visual.spec.ts

# Run with specific browser
npm run playwright:test --project=chromium
```

### Debugging Tests

1. **View test logs:**
   ```bash
   npm run playwright:logs
   ```

2. **Run with headed browser locally:**
   ```bash
   npx playwright test --headed
   ```

3. **Debug mode:**
   ```bash
   npx playwright test --debug
   ```

## Visual Testing

### Updating Screenshots

When UI changes require new visual baselines:

```bash
# Update all visual snapshots
npm run playwright:update-snapshots

# Update specific test snapshots
npm run playwright:test --update-snapshots tests/visual.spec.ts
```

### Screenshot Organization

Screenshots are stored in `tests/visual.spec.ts-snapshots/` with naming pattern:
- `{test-name}-{browser}-{platform}.png`
- Example: `main-page-full-chromium-linux.png`

## Troubleshooting

### Common Issues

1. **Port conflicts:**
   ```bash
   # Check what's using port 3000 or 5000
   lsof -i :3000
   lsof -i :5000
   
   # Stop conflicting services
   npm run server:down
   docker stop $(docker ps -q)
   ```

1.1. **Port 5000 conflict with AirPlay Receiver**

When starting the Playwright server, MacOS users can figure the following error: `Error response from daemon: Ports are not available: exposing port TCP 0.0.0.0:5000 -> 127.0.0.1:0: listen tcp 0.0.0.0:5000: bind: address already in use`, which can be a conflict with the AirPlay Receiver, to fix it:

- Go to System Settings > AirDrop & Handoff
- Turn off AirPlay Receiver
- This will free up port 5000

2. **Grafana not starting:**
   ```bash
   # Check Grafana logs
   docker compose logs grafana
   
   # Rebuild Grafana container
   docker compose up --build -d
   ```

3. **Tests failing with connection errors:**
   ```bash
   # Verify Grafana health
   curl http://localhost:3000/api/health
   
   # Check if plugin is loaded
   curl http://localhost:3000/api/plugins/grafana-llm-app
   ```

4. **Visual test failures:**
   - Run `npm run playwright:update-snapshots` to regenerate baselines
   - Check if tests are running in different environments (Docker vs local)

### Environment Variables

- `GRAFANA_BASE_URL` - Override Grafana URL (default: auto-detected)
- `PW_TEST_CONNECT_WS_ENDPOINT` - Playwright server endpoint
- `PW_TEST_HTML_REPORT_OPEN` - Control HTML report behavior

## CI Integration

For continuous integration, use the automated commands:

```yaml
# GitHub Actions example
- name: Run E2E Tests
  run: npm run test:e2e-full
```

The automated workflow:
1. Builds plugin (frontend + backend)
2. Starts Grafana with plugin mounted
3. Waits for Grafana health check
4. Runs all tests in Docker container
5. Collects results and artifacts
6. Cleans up all services

## Performance

- **Test execution time:** ~15-20 seconds for full suite
- **Container startup:** ~10-15 seconds for Grafana + Playwright
- **Network isolation:** All services communicate via Docker network (no external dependencies)

## CI Integration

The e2e tests are integrated into GitHub Actions workflows for automated testing:

### Workflows

1. **Plugin Release** (`plugin-release.yml`)
   - Runs e2e tests during release builds
   - Tests must pass before plugin validation
   - Uploads test artifacts on failure

2. **Pull Request Testing** (`run-tests.yml`)
   - Runs e2e tests on every PR
   - Provides early feedback on changes
   - Shorter artifact retention (3 days)

3. **Dedicated E2E Testing** (`e2e-tests.yml`)
   - Manual trigger with configurable options
   - Nightly scheduled runs
   - Multi-browser testing support
   - Visual snapshot updates

### Workflow Features

- **Multi-browser testing:** Chrome, Firefox, Safari/WebKit
- **Grafana version selection:** Test against different Grafana versions
- **Snapshot management:** Update visual test baselines in CI
- **Artifact collection:** Test results, reports, and screenshots
- **Automatic cleanup:** Proper Docker container management
- **Failure reporting:** Detailed logs and GitHub comments

### Manual Workflow Dispatch

To run e2e tests manually:

1. Go to **Actions** → **E2E Tests** in GitHub
2. Click **Run workflow**
3. Configure options:
   - **Browser:** chromium, firefox, webkit, or all
   - **Update Snapshots:** true/false
   - **Grafana Version:** main, latest, or specific version

### CI Environment Variables

The workflows use these environment variables:
- `DOCKER_BUILDKIT=1` - Enable Docker BuildKit
- `COMPOSE_DOCKER_CLI_BUILD=1` - Use Docker Compose CLI
- `GRAFANA_VERSION` - Override Grafana version
- `PLAYWRIGHT_PROJECT` - Specify browser for testing

### Artifacts

CI workflows generate these artifacts:
- **Test Results:** Raw Playwright test output
- **HTML Reports:** Interactive test reports with screenshots
- **Visual Snapshots:** Updated baseline images
- **Container Logs:** Debug information for failures

### Status Checks

- ✅ **Required:** e2e tests must pass in plugin-release workflow
- ⚠️ **Optional:** e2e tests in PR workflow (informational)
- 📊 **Reporting:** Automatic GitHub comments with results

## Best Practices

1. **Always clean up:** Use provided scripts to stop services
2. **Run specific tests during development:** Avoid full suite for quick iteration
3. **Update snapshots deliberately:** Only when UI changes are intentional
4. **Use test IDs consistently:** Follow the established pattern in `testIds.ts`
5. **Test real functionality:** Focus on user workflows, not implementation details
6. **Monitor CI results:** Check GitHub Actions for automated test feedback
7. **Use manual workflows:** Run specific browser tests when needed
8. **Update snapshots in CI:** Use the dedicated workflow for visual updates