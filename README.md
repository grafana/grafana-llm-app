# LLM Plugin and frontend libraries repo

This repository holds separate packages for Grafana LLM Plugin and the `@grafana/llm` package for interfacing with it.

They are placed into a monorepo because they are tighlty coupled, and should be developed together with identical dependencies where possible.

Each package has its own package.json with its own npm commands, but the intention is that you should be able to run and test both of them from root.

## Contributing

If you're interested in adding support for new LLM providers or extending the functionality of the Grafana LLM App, please see the [CONTRIBUTING.md](./CONTRIBUTING.md) file for detailed implementation guidance.

## Developing these packages

### Quickstart

1. Install node >= 22, go >= 1.21, and [Mage](https://magefile.org/).
2. Install docker
3. Run `npm install`
4. Run `npm run dev`
5. In a separate terminal from dev, run `npm run server`
If you want to bring up the vector services, you can run `COMPOSE_PROFILES=vector npm run server` instead

6. Go to (http://localhost:3000/plugins/grafana-llm-app) to see configuration page and the developer sandbox

This will watch the frontend dependencies and update live. If you are changing backend dependencies, you can run `npm run backend:restart` to rebuild the backend dependencies and restart the plugin in the Grafana server.

### Dev Sandbox

It is recommended to develop functionality against the "developer sandbox", only available when the app is run in dev mode, that can be opened from the configuration page, because it gives you a clean end to end test of any changes. This can be used for modifications to the @grafana/llm library, the backend plugin, or functionality built on top of those packages.

### Backend

It is recommended to run these using npm commands from root so that the entire project can be developed from the root directory. If you want to dig into the commands themselves, you can read the corresponding scripts in ./packages/grafana-llm-app/package.json

1. Update [Grafana plugin SDK for Go](https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins/grafana-plugin-sdk-for-go) dependency to the latest minor version:

   ```bash
   npm run backend:update-sdk
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   npm run backend:build
   ```

3. Test the backend

   ```bash
   npm run backend:test
   ```

4. To see all available Mage targets for additional commands, run this from ./packages/grafana-llm-app/:

   ```bash
   mage -l
   ```

### Frontend

1. Install dependencies

   ```bash
   npm install
   ```

2. Build plugin in development mode and run in watch mode

   ```bash
   npm run dev
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes, requires git init first
   npm run test

   # Exits after running all the tests
   npm run test:ci
   ```

5. Run end-to-end tests (using Playwright)

   ```bash
   # Run e2e tests (builds the plugin, starts containers, runs tests, cleans up)
   npm run test:e2e

   # Run e2e tests with full build (same as above but explicitly builds first)
   npm run test:e2e-full

   # Run e2e tests in CI mode (with optimized Docker builds)
   npm run test:e2e-ci
   ```

   **Note:** The e2e tests use Docker containers and require the workspace to be built first. The tests use a special `SKIP_PREINSTALL=true` environment variable in the Docker containers to prevent npm installation conflicts with the workspace's preinstall script.

6. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```

## Release process

### Plugin Release
- Bump version in `packages/grafana-llm-app/package.json` (e.g., 0.2.0 to 0.2.1)
- Add notes to changelog describing changes since last release
- Merge PR for a branch containing those changes into main
- Trigger "Plugin Release" from the Actions tab on Github.

### NPM Release
This is for the npm package `@grafana/llm`.
- Bump version in `packages/grafana-llm-frontend/package.json` (e.g., 0.2.0 to 0.2.1)
- Tip: You can do this in the same PR as the Plugin Release one above.
- Trigger "NPM Release" action from the Actions tab on Github.

### llmclient Release
- Push a new tag to the repo (e.g., `git tag -a llmclient/v0.X.X -m "llmclient v0.X.X release"`)
