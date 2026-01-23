# Changelog

## [1.3.0](https://github.com/itsmostafa/goralph/compare/v1.2.0...v1.3.0) (2026-01-23)


### Features

* **cli:** add CLI provider type and --cli flag ([90dce37](https://github.com/itsmostafa/goralph/commit/90dce371198e373feea70d132b3bfde714053e2c))
* **cli:** add OpenAI Codex CLI support ([e591048](https://github.com/itsmostafa/goralph/commit/e5910486ed066ad8907538b12a640aa1be02f45d))
* **cmd:** generate unique plan file per session ([87435ca](https://github.com/itsmostafa/goralph/commit/87435ca5a419c0fdd115e907562a649c8c206cf8))
* **codex:** implement Codex CLI output parsing ([d2e2069](https://github.com/itsmostafa/goralph/commit/d2e2069bc01357499cf84606fea8165e91cc8fe6))
* **loop:** add completion promise detection for early loop termination ([b0dfdc7](https://github.com/itsmostafa/goralph/commit/b0dfdc7779551ec109f4e449d3bca0b1a99fc1c1))
* **loop:** add completion promise detection for early loop termination ([6538782](https://github.com/itsmostafa/goralph/commit/65387820c27a809493590ae30298f49439e10c36))
* **loop:** add Provider interface for CLI abstraction ([b6883bd](https://github.com/itsmostafa/goralph/commit/b6883bd3b435d72b0d2959e2cae1fc4293620d1a))
* **loop:** add session-scoped implementation plan files ([08bd675](https://github.com/itsmostafa/goralph/commit/08bd675fe01b6530434015f85f3923c48935e557))
* **loop:** add session-scoped plan path generation ([01218c4](https://github.com/itsmostafa/goralph/commit/01218c44c3d62ef743053ee45920c28fc375c595))
* **loop:** add support for providers without cost/duration data ([326f024](https://github.com/itsmostafa/goralph/commit/326f024e7ef2145fb6f521237a60f3134d4c74be))
* **loop:** display model in header for Claude and Codex providers ([9ad405b](https://github.com/itsmostafa/goralph/commit/9ad405bb413631733b387c81fb1539a15886b3df))
* **loop:** use session-scoped plan files in loop execution ([4ce5144](https://github.com/itsmostafa/goralph/commit/4ce5144d6dcb28bf284be7b7a7a3b92719e329ff))


### Bug Fixes

* **loop:** count reasoning items as turns for Codex provider ([616f8f3](https://github.com/itsmostafa/goralph/commit/616f8f39f275582ae6fdf0bbeabeab5c57d972fb))
* **loop:** use millisecond precision in plan path to prevent collisions ([35451d7](https://github.com/itsmostafa/goralph/commit/35451d77a28301ec9687d4b5edcb0296db185ff4))

## [1.2.0](https://github.com/itsmostafa/goralph/compare/v1.1.0...v1.2.0) (2026-01-22)


### Features

* **cli:** add --no-push flag to skip pushing after iterations ([1b6b549](https://github.com/itsmostafa/goralph/commit/1b6b549c6b162e26fb2e8ee74bd2ba23d4923e4f))
* **cli:** add --no-push flag to skip pushing after iterations ([5cfd41f](https://github.com/itsmostafa/goralph/commit/5cfd41f8d091fefa436e54bb771135d30e980398))

## [1.1.0](https://github.com/itsmostafa/goralph/compare/v1.0.0...v1.1.0) (2026-01-22)


### Features

* **loop:** add implementation plan support with Codex CLI integration plan ([5f2dad0](https://github.com/itsmostafa/goralph/commit/5f2dad0719583f1b352bb9f3dd1b60ff2bc620e2))
* **loop:** add iteration-aware task generation guidance ([939affa](https://github.com/itsmostafa/goralph/commit/939affa6aaa85e428a76635a4535b06aaad085a5))
* **loop:** add iteration-aware task generation guidance ([4b16afd](https://github.com/itsmostafa/goralph/commit/4b16afdfe6c90c98de1b67a989800151305bc48b))
* **runner:** add Runner interface for AI CLI abstraction ([77e2496](https://github.com/itsmostafa/goralph/commit/77e249699bfebfc38cbaac64826e8123d0e0d5d8))
* **version:** add version package with ldflags support ([6742732](https://github.com/itsmostafa/goralph/commit/6742732e596dee010452e7a0d4a8915fedeee36e))

## 1.0.0 (2026-01-22)


### Features

* add Go application scaffolding ([bc5f2a1](https://github.com/itsmostafa/goralph/commit/bc5f2a16f938989c7bc1a7c97b031bb2341f261b))
* **cli:** add cobra CLI with build and plan commands ([0f4d17e](https://github.com/itsmostafa/goralph/commit/0f4d17ecc77a2cec711c61fdc3862f30dcbc7dfc))
* **loop:** add agentic loop script for Ralph Wiggum technique ([82d2397](https://github.com/itsmostafa/goralph/commit/82d2397979e8df81b5047005c21bad5b2eb74e25))
* **loop:** add JSON message types for Claude output parsing ([413a64e](https://github.com/itsmostafa/goralph/commit/413a64e149d416e64c374fccff64581e5bdcc449))
* **loop:** add JSON parsing and iteration logging ([2721aee](https://github.com/itsmostafa/goralph/commit/2721aeebad2db3278709728beed2734c5dc2609a))
* **loop:** add real-time streaming output with tool status indicators ([66e5148](https://github.com/itsmostafa/goralph/commit/66e514845811ac19bcc25071987faf7948c228ef))
* **loop:** add styled output formatting with lipgloss ([7f1f78a](https://github.com/itsmostafa/goralph/commit/7f1f78a71ea0af0f219d6e495fb89ad8e4b72d1b))
* **taskfile:** add install task for ~/.local/bin ([87d7d7e](https://github.com/itsmostafa/goralph/commit/87d7d7e5012ca9f591908e8d0f774d8da253a481))
