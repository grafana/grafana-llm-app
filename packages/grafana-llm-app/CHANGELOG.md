# Changelog

## Unreleased

- Update Anthropic model defaults by @joe-elliott
  - Base: `claude-sonnet-4-20250514`
  - Large: `claude-sonnet-4-20250514`

## 0.22.1

- feat: relax minimum Grafana version requirement by @sd2k in #724

## 0.22.0

- feat: add MCP StreamableHTTP server to /mcp endpoint of LLM app by @sd2k in #691
- fix: MCP isEnabled check and improve back-compatibility by @sd2k in #717

## 0.21.2

- fix: handle case when nested provider differs by @sd2k in #704
- feat: add log line when MCP stream can't be found by @sd2k in #703

## 0.21.1

- fix: revert token timeout to 10 min by @annanay25 in #698

## 0.21.0

- feat: export 'enabled' function from mcp module of npm package by @sd2k in #689
- feat: refresh Grafana access token before expiry in Cloud by @sd2k in #688
- refactor: add MCP struct and simplify access token usage by @sd2k in #696

## 0.20.1

- feat: increase OBO user auth token TTL to 30 min by @annanay25 in #684

## 0.20.0

- feat: remove MCP feature flag and enable MCP by default by @annanay25 in #670

## 0.19.3

- feat: upgrade MCP Grafana integration to v0.4.0 by @sd2k in #673

## 0.19.2

- feat: remove OpenAI Assistant specific code by @csmarchbanks in #663
- feat: enable LLM app to create access tokens on-behalf-of user by @annanay25 in #644

## 0.19.1

- feat: add Dashboard and Sift tools by @csmarchbanks in #661
- feat: allow customizing the OpenAI API path in the app config page by @sd2k in #656

## 0.19.0

- feat: upgrade MCP Grafana integration to v0.3.0 by @csmarchbanks in #658
- feat: add asserts tool by @xujiaxj in #657

## 0.18.0

- feat: add enabled flag to MCPProvider context value and fix infinite loop when MCP is not enabled by @sd2k in #648

## 0.17.0

- bug: Make sure at least one message has the user role for Anthropic by @edwardcqian in #629
- Update model defaults by @csmarchbanks in #632 and #635:
  - Base: `gpt-4o-mini` to `gpt-4.1-mini`
  - Large: `gpt-4o` to `gpt-4.1`

## 0.16.0

- chore: bump github.com/grafana/mcp-grafana to 0.2.4 by @sd2k in #618. This adds more tools to retrieve dashboards and OnCall details.
- Switch to Anthropic's OpenAI-compatible API by @gitdoluquita in #617. This PR also adds helper functions to execute tool calls with streaming endpoints.

## 0.15.0

- removed mentions for public preview from README.md by @Maurice-L-R in #610
- Bump a full minor version to do a release so we can publish a version out of public preview by @SandersAaronD in #612

## 0.14.1

- fix: improve dev sandbox prompt by @sd2k in #598
- workaround: send publish messages over Websocket by @sd2k in #601
- fix: use public API to publish, where possible by @sd2k in #603
- fix: check for >= 3 args in publish, not === 3 by @sd2k in #604
- chore: bump github.com/grafana/mcp-grafana to 0.2.2 by @sd2k in #605
- chore: add CODEOWNERS by @sd2k in #607

## 0.14.0

- Allow enabling of dev sandbox via jsonData by @csmarchbanks in #595
- feat: run MCP server in backend using Grafana Live by @sd2k in #574
- feat: add MCP helper functions to the @grafana/llm package by @sd2k in #590

## 0.13.2

- docs: Add guide for implementing new LLM providers by @gitdoluquita in #573
- Bug: duplicate healthcheck details for backwards compatiblity by @edwardcqian in #585

## 0.13.1

- update action versions in plugin-release by @csmarchbanks in #577
- fix: update build:all command to also build plugin frontend by @sd2k in #578

## 0.12.0

- Add support for OpenAI assistant functionality
- Upgrade various dependencies

## 0.11.0

- Update model defaults:
  - Base: `gpt-3.5-turbo` to `gpt-4o-mini`
  - Large: `gpt-4-turbo` to `gpt-4o`

## 0.10.9

- Update wording around OpenAI and open models
- Update LLM setup instructions

## 0.10.8

- Bump various dependencies to fix security issues (e.g. #446)

## 0.10.7

- Use non-crypto UUID generation for stream request #409

## 0.10.6

- Fix bug where it was impossible to fix a saved invalid OpenAI url (#405)

## 0.10.1

- Settings: differentiate between disabled and not configured (#350)

## 0.10.0

- Breaking: use `base` and `large` model names instead of `small`/`medium`/`large` (#334)
- Breaking: remove function calling arguments from `@grafana/llm` package (#343)
- Allow customisation of mapping between abstract model and provider model, and default model (#337, #338, #340)
- Make the `model` field optional for chat completions & chat completion stream endpoints (#341)
- Don't preload the plugin to avoid slowing down Grafana load times (#339)

## 0.9.1

- Fix handling of streaming requests made via resource endpoints (#326)

## 0.9.0

- Initial backend support for abstracted models (#315)

## 0.8.6

- Fix panic with stream EOF (#308)

## 0.8.5

- Added a `displayVectorStoreOptions` flag to optionally display the vector store configs

## 0.8.1

- Add mitigation for side channel attacks

## 0.7.0

- Refactors repo into monorepo together with frontend dependencies
- Creates developer sandbox for developing frontend dependencies
- Switches CI/CD to github actions

## 0.6.4

- Fix bug where resource calls to OpenAI would fail for Grafana managed LLMs

## 0.6.3

- Fix additional UI bugs
- Fix issue where health check returned true even if LLM was disabled

## 0.6.2

- Fix UI issues around OpenAI provider introduced in 0.6.1

## 0.6.1

- Store Grafana-managed OpenAI opt-in in ML cloud backend DB (Grafana Cloud only)
- Updated Grafana-managed OpenAI opt-in messaging (Grafana Cloud only)
- UI update for LLM provider selection

## 0.6.0

- Add Grafana-managed OpenAI as a provider option (Grafana Cloud only)

## 0.5.2

- Allow Qdrant API key to be configured in config UI, not just when provisioning

## 0.5.1

- Fix issue where temporary errors were cached, causing /health to fail permanently.

## 0.5.0

- Add basic auth to VectorAPI

## 0.4.0

- Add 'Enabled' switch for vector services to configuration UI
- Added instructions for developing with example app
- Improve health check to return more granular details
- Add support for filtered vector search
- Improve vector service health check

## 0.3.0

- Add Go package providing an OpenAI client to use the LLM app from backend Go code
- Add support for Azure OpenAI. The plugin must be configured to use OpenAI and provide a link between OpenAI model names and Azure deployment names
- Return streaming errors as part of the stream, with objects like `{"error": "<error message>"}`

## 0.2.1

- Improve health check endpoint to include status of various features
- Change path handling for chat completions streams to put separate requests into separate streams. Requests can pass a UUID as the suffix of the path now, but is backwards compatible with an older version of the frontend code.

## 0.2.0

- Expose vector search API to perform semantic search against a vector database using a configurable embeddings source

## 0.1.0

- Support proxying LLM requests from Grafana to OpenAI
