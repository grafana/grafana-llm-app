# LLM Plugin and frontend libraries repo

This repository holds separate packages for Grafana LLM Plugin and the `@grafana/llm` package for interfacing with it.

They are placed into a monorepo because they are tighlty coupled, and should be developed together with identical dependencies where possible.

Each package has its own package.json with its own npm commands, but the intention is that you should be able to run and test both of them from root.

## Developing these packages

### Quickstart

1. Install node >= 20, go >= 1.21, and [Mage](https://magefile.org/).
2. Install docker
3. Run `npm install`
4. Run `npm run dev`
5. In a separate terminal from dev, run `npm run server`
6. Go to (http://localhost:3000/plugins/grafana-llm-app) to see configuration page and the developer sandbox

This will watch the frontend dependencies and update live. If you are changing backend dependencies you will need to exit from `npm run dev` and `npm run server` and repeat those steps, because the backend dependencies do not build or update within the server automatically.

### Dev Sandbox

It is recommended to develop functionality against the "developer sandbox", only available when the app is run in dev mode, that can be opened from the configuration page, because it gives you a clean end to end test of any changes. This can be used for modifications to the @grafana/llm library, the backend plugin, or functionality built on top of those packages.

### Backend

It is recommended to run these using npm commands from root so that the entire project can be developed from the root directory. If you want to dig into the commands themselves, you can read the corresponding scripts in ./packages/grafana-llm-app/package.json

1. Update [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/) dependency to the latest minor version:

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

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Cypress)

   ```bash
   # Spins up a Grafana instance first that we tests against
   npm run e2e:ci
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```

## Release process

### Plugin Release
- Bump version in package.json (e.g., 0.2.0 to 0.2.1)
- Bump version in `packages/grafana-llm-app/package.json` (e.g., 0.2.0 to 0.2.1)
- Add notes to changelog describing changes since last release
- Merge PR for a branch containing those changes into main

### llmclient Release
- Push a new tag to the repo (e.g., `git tag -a llmclient/v0.X.X -m "llmclient v0.X.X release"`)
