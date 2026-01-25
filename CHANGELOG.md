# Changelog

## [0.1.2](https://github.com/itsmostafa/goralph/compare/v0.1.1...v0.1.2) (2026-01-25)


### Features

* **cmd:** add CLI flags for RLM mode and verification ([4f5ecf2](https://github.com/itsmostafa/goralph/commit/4f5ecf223d5079dcda4653e06d1f369222528677))
* **cmd:** replace --rlm flag with --mode flag ([9d91b47](https://github.com/itsmostafa/goralph/commit/9d91b475c59a7e3cf85d79e857973e3ff1131e8c))
* **loop:** add Mode type and ModeRunner interface ([63e960d](https://github.com/itsmostafa/goralph/commit/63e960db0bf7cbc7ea24347f29aae794d00e650a))
* **loop:** add PhaseRouter for RLM phase inference and guidance ([53f922e](https://github.com/itsmostafa/goralph/commit/53f922ed5ecccf183ef07204d9bd2773e700d858))
* **loop:** add RalphRunner implementing ModeRunner ([f559c53](https://github.com/itsmostafa/goralph/commit/f559c5317f9dbf2facccb8149ba7ce5f4ec8f5fd))
* **loop:** add RLM mode with structured phase-based execution ([1734469](https://github.com/itsmostafa/goralph/commit/1734469151e3221859f5936257131682c5071e4d))
* **loop:** add RLM output formatting ([9b07d25](https://github.com/itsmostafa/goralph/commit/9b07d25a336ba17b3f61fd08338c23cf5a44a80f))
* **loop:** add RLM prompt builder with context injection ([ef97dcf](https://github.com/itsmostafa/goralph/commit/ef97dcf0ae4507340dd9e9c61cb4a268456480ff))
* **loop:** add RLM type definitions and constants ([21804f4](https://github.com/itsmostafa/goralph/commit/21804f4eb839c4724f2d216fb0d92eb9eadec2f4))
* **loop:** add RLMRunner implementing ModeRunner ([564cac7](https://github.com/itsmostafa/goralph/commit/564cac79385c6bef64529f19b1f33aa9ab210cc1))
* **loop:** add StateManager for RLM state persistence ([24ce92e](https://github.com/itsmostafa/goralph/commit/24ce92e8085bf4a78540c995ff987d3fdd009817))
* **loop:** add Verifier for build/test validation ([33196b7](https://github.com/itsmostafa/goralph/commit/33196b7202887c221e8fe717e8259c76eff60e41))
* **loop:** detect RLM markers in agent output ([975a755](https://github.com/itsmostafa/goralph/commit/975a755580fe9f273b8b318edf2960071dc05271))
* **loop:** extend Config with RLM and verification options ([149564c](https://github.com/itsmostafa/goralph/commit/149564cddf4edafb89be24b8eba63525e8cec69e))
* **loop:** integrate RLM mode and verification into main loop ([1a33b1e](https://github.com/itsmostafa/goralph/commit/1a33b1e616bdbceca685bf7a68959cd152a6c871))
* **loop:** introduce ModeRunner interface for extensible execution modes ([989bb18](https://github.com/itsmostafa/goralph/commit/989bb18f098ca78f49ebaa36c6d4910cd2d7ee67))
* **loop:** make --no-push skip commits in addition to pushes ([5a2cac0](https://github.com/itsmostafa/goralph/commit/5a2cac00b3f4b5783030cd3198f03d1c518572b6))
* **output:** update tool completion indicators in-place using ANSI cursor control ([bfca20d](https://github.com/itsmostafa/goralph/commit/bfca20d229a8da9511a8aa48bd1f410dd92fc9f0))
* **rlm:** add JSON schemas for agent-written state files ([fcb9aa6](https://github.com/itsmostafa/goralph/commit/fcb9aa65d53d93574063a6826c07b0d50dfc7ebc))


### Bug Fixes

* **loop:** reset text tracking after tool results to prevent truncation ([8f580db](https://github.com/itsmostafa/goralph/commit/8f580dbcdc52cd1b39a1e95c8b2b6ef3283847db))
* **loop:** run --verify without requiring RLM marker ([04c8255](https://github.com/itsmostafa/goralph/commit/04c8255cb7a14ba875ca2551753cf4c301fafcbb))
* **loop:** start first iteration in PLAN phase ([7ca2052](https://github.com/itsmostafa/goralph/commit/7ca2052208e9dfff42aab36d6df165c3115699b3))
* **loop:** use zero-padded iteration in state filenames ([35926bc](https://github.com/itsmostafa/goralph/commit/35926bc88babb0cd36844a8f3bfa468165bb3ed2))
* **rlm:** ensure agent implements during ACT phase ([4a9b57c](https://github.com/itsmostafa/goralph/commit/4a9b57c45ae33b28e7d7c2535dec0a0f111d1976))
* **rlm:** explicitly instruct agent to implement changes in no-push mode ([2e35a73](https://github.com/itsmostafa/goralph/commit/2e35a739c2f4594ca793f49368727c0470dd6f7a))

## [0.1.1](https://github.com/itsmostafa/goralph/compare/v0.1.0...v0.1.1) (2026-01-23)


### Bug Fixes

* **output:** improve newline handling between text and tool indicators ([2ba5eb8](https://github.com/itsmostafa/goralph/commit/2ba5eb85cdf6d48a3272d67bb6d4fac4d81ae2ce))
* **output:** improve newline handling between text and tool indicators ([ce823c6](https://github.com/itsmostafa/goralph/commit/ce823c6523a6fef4979cfa46d861d9ec1079e43d))

## 0.1.0 (2026-01-23)


### ⚠ BREAKING CHANGES

* **cli:** The 'build' and 'plan' subcommands have been removed. Use 'goralph run' instead.

### Features

* add Go application scaffolding ([bc5f2a1](https://github.com/itsmostafa/goralph/commit/bc5f2a16f938989c7bc1a7c97b031bb2341f261b))
* **cli:** add --no-push flag to skip pushing after iterations ([1b6b549](https://github.com/itsmostafa/goralph/commit/1b6b549c6b162e26fb2e8ee74bd2ba23d4923e4f))
* **cli:** add --no-push flag to skip pushing after iterations ([5cfd41f](https://github.com/itsmostafa/goralph/commit/5cfd41f8d091fefa436e54bb771135d30e980398))
* **cli:** add CLI provider type and --cli flag ([90dce37](https://github.com/itsmostafa/goralph/commit/90dce371198e373feea70d132b3bfde714053e2c))
* **cli:** add cobra CLI with build and plan commands ([0f4d17e](https://github.com/itsmostafa/goralph/commit/0f4d17ecc77a2cec711c61fdc3862f30dcbc7dfc))
* **cli:** add OpenAI Codex CLI support ([e591048](https://github.com/itsmostafa/goralph/commit/e5910486ed066ad8907538b12a640aa1be02f45d))
* **cmd:** generate unique plan file per session ([87435ca](https://github.com/itsmostafa/goralph/commit/87435ca5a419c0fdd115e907562a649c8c206cf8))
* **codex:** implement Codex CLI output parsing ([d2e2069](https://github.com/itsmostafa/goralph/commit/d2e2069bc01357499cf84606fea8165e91cc8fe6))
* **loop:** add agentic loop script for Ralph Wiggum technique ([82d2397](https://github.com/itsmostafa/goralph/commit/82d2397979e8df81b5047005c21bad5b2eb74e25))
* **loop:** add completion promise detection for early loop termination ([b0dfdc7](https://github.com/itsmostafa/goralph/commit/b0dfdc7779551ec109f4e449d3bca0b1a99fc1c1))
* **loop:** add completion promise detection for early loop termination ([6538782](https://github.com/itsmostafa/goralph/commit/65387820c27a809493590ae30298f49439e10c36))
* **loop:** add implementation plan support with Codex CLI integration plan ([5f2dad0](https://github.com/itsmostafa/goralph/commit/5f2dad0719583f1b352bb9f3dd1b60ff2bc620e2))
* **loop:** add iteration-aware task generation guidance ([939affa](https://github.com/itsmostafa/goralph/commit/939affa6aaa85e428a76635a4535b06aaad085a5))
* **loop:** add iteration-aware task generation guidance ([4b16afd](https://github.com/itsmostafa/goralph/commit/4b16afdfe6c90c98de1b67a989800151305bc48b))
* **loop:** add JSON message types for Claude output parsing ([413a64e](https://github.com/itsmostafa/goralph/commit/413a64e149d416e64c374fccff64581e5bdcc449))
* **loop:** add JSON parsing and iteration logging ([2721aee](https://github.com/itsmostafa/goralph/commit/2721aeebad2db3278709728beed2734c5dc2609a))
* **loop:** add Provider interface for CLI abstraction ([b6883bd](https://github.com/itsmostafa/goralph/commit/b6883bd3b435d72b0d2959e2cae1fc4293620d1a))
* **loop:** add real-time streaming output with tool status indicators ([66e5148](https://github.com/itsmostafa/goralph/commit/66e514845811ac19bcc25071987faf7948c228ef))
* **loop:** add session-scoped implementation plan files ([08bd675](https://github.com/itsmostafa/goralph/commit/08bd675fe01b6530434015f85f3923c48935e557))
* **loop:** add session-scoped plan path generation ([01218c4](https://github.com/itsmostafa/goralph/commit/01218c44c3d62ef743053ee45920c28fc375c595))
* **loop:** add styled output formatting with lipgloss ([7f1f78a](https://github.com/itsmostafa/goralph/commit/7f1f78a71ea0af0f219d6e495fb89ad8e4b72d1b))
* **loop:** add support for providers without cost/duration data ([326f024](https://github.com/itsmostafa/goralph/commit/326f024e7ef2145fb6f521237a60f3134d4c74be))
* **loop:** display model in header for Claude and Codex providers ([9ad405b](https://github.com/itsmostafa/goralph/commit/9ad405bb413631733b387c81fb1539a15886b3df))
* **loop:** use session-scoped plan files in loop execution ([4ce5144](https://github.com/itsmostafa/goralph/commit/4ce5144d6dcb28bf284be7b7a7a3b92719e329ff))
* **runner:** add Runner interface for AI CLI abstraction ([77e2496](https://github.com/itsmostafa/goralph/commit/77e249699bfebfc38cbaac64826e8123d0e0d5d8))
* **taskfile:** add install task for ~/.local/bin ([87d7d7e](https://github.com/itsmostafa/goralph/commit/87d7d7e5012ca9f591908e8d0f774d8da253a481))
* **version:** add version package with ldflags support ([6742732](https://github.com/itsmostafa/goralph/commit/6742732e596dee010452e7a0d4a8915fedeee36e))


### Bug Fixes

* **loop:** count reasoning items as turns for Codex provider ([616f8f3](https://github.com/itsmostafa/goralph/commit/616f8f39f275582ae6fdf0bbeabeab5c57d972fb))
* **loop:** use millisecond precision in plan path to prevent collisions ([35451d7](https://github.com/itsmostafa/goralph/commit/35451d77a28301ec9687d4b5edcb0296db185ff4))
* move bump-minor-pre-major inside package config ([d8e1d68](https://github.com/itsmostafa/goralph/commit/d8e1d683165f63bcdd712560883e3161ae07b629))
* prevent breaking changes from bumping to 1.0.0 ([21ef1d6](https://github.com/itsmostafa/goralph/commit/21ef1d6be1dace2c0f2f6c29c946264ed979a29e))


### Code Refactoring

* **cli:** replace build/plan commands with unified run command ([80a561a](https://github.com/itsmostafa/goralph/commit/80a561ab46f3e339609fd9dd8a31dd1b466ba842))

## Changelog
